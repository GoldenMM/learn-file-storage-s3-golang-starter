package main

import (
	"fmt"
	"io"
	"net/http"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

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

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	// TODO: implement the upload here

	// Set a max memory and parse the form
	maxMemory := int64(10 << 20) // 10 MB
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse form", err)
		return
	}

	// Get the file from the form and get the content type
	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't parse form", err)
		return
	}
	defer file.Close()
	contentType := header.Header.Get("Content-Type")

	// Convert the file to a byte array
	fileData, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't read file", err)
		return
	}

	// Get the video's meta-data from the database
	dbVideo, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video", err)
		return
	}
	// Check if the video belongs to the user
	if dbVideo.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "You don't own this video", nil)
		return
	}
	// Save the file to the global map
	videoThumbnails[videoID] = thumbnail{
		data:      fileData,
		mediaType: contentType,
	}
	// Save the new thumbnail to the database
	stringVideoID := videoID.String()
	thumbnailURL := fmt.Sprintf("http://localhost:%v/api/thumbnails/%s", cfg.port, stringVideoID)
	fmt.Println("thumbnailURL: ", thumbnailURL)

	dbVideo.ThumbnailURL = &thumbnailURL

	err = cfg.db.UpdateVideo(dbVideo)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}
	// Respond with the videos meta-data
	respondWithJSON(w, http.StatusOK, dbVideo)

}
