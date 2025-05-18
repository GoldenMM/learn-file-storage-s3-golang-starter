package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {

	// Set an upload limit
	maxUpload := int64(1 << 30) // 1 GB
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)

	// Get the video ID from the URL
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	// Authenticate the user and get the user ID
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}
	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}
	fmt.Println("uploading video", videoID, "by user", userID)

	// Get the video metadata from the database
	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video", err)
		return
	}
	if dbVideo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You don't own this video", nil)
		return
	}

	// Set a max memory and parse the form
	err = r.ParseMultipartForm(maxUpload)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Max Memory exceded", err)
		return
	}
	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse form", err)
		return
	}
	defer file.Close()

	// Check if the content type is video/mp4
	contentType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse content type", err)
		return
	}
	if contentType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid content type", nil)
		return
	}

	// Save the video to a temporary file
	tmpFile, err := os.CreateTemp("", "tubely-upload-*.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create temporary file", err)
		return
	}

	_, err = io.Copy(tmpFile, file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't save video", err)
		return
	}
	// Reset the file pointer to the beginning
	_, err = tmpFile.Seek(0, io.SeekStart)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't seek to beginning of file", err)
		return
	}
	defer tmpFile.Close()

	// Generate a unique file name
	videoFileID := make([]byte, 32)
	_, err = rand.Read(videoFileID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't generate random bytes", err)
		return
	}

	// Check if the file exists
	if _, err := os.Stat(tmpFile.Name()); err != nil {
		fmt.Println("Temp file stat error:", err)
	}

	// Get the aspect ratio of the video
	aspectRatio, err := getVideoAspectRatio(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video aspect ratio", err)
		return
	}
	// Set the subdirectory for the video based on the aspect ratio
	var subdirectory string
	switch aspectRatio {
	case "16:9":
		subdirectory = "landscape"
	case "9:16":
		subdirectory = "portrait"
	default:
		subdirectory = "other"
	}

	// Process the video for fast start
	processedFileIDString, err := processVideoForFastStart(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't process video for fast start", err)
		return
	}
	// Close the temporary file
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// open the processed file
	processedFile, err := os.Open(processedFileIDString)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't open processed file", err)
		return
	}
	defer processedFile.Close()

	// Convert the processedFileID to a string and adds the file extension
	videoFileIDString := subdirectory + "/" + base64.RawURLEncoding.EncodeToString([]byte(videoFileID)) + ".mp4"

	// Put the file in S3
	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(cfg.s3Bucket),
		Key:         aws.String(videoFileIDString),
		Body:        processedFile,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't upload video", err)
		return
	}

	// Update the video metadata in the database
	videoURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
		cfg.s3Bucket,
		cfg.s3Region,
		videoFileIDString)
	dbVideo.VideoURL = &videoURL
	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}
	// Respond with the video URL
	respondWithJSON(w, http.StatusOK, dbVideo)

}
