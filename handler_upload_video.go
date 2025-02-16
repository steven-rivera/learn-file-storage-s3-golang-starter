package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadVideo(w http.ResponseWriter, r *http.Request) {
	const uploadLimit = 1 << 30 // 1 Gibibyte (2^30)
	r.Body = http.MaxBytesReader(w, r.Body, uploadLimit)


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

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't find video", err)
		return 
	}
	if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Not authorized to update this video", err)
		return
	}

	file, header, err := r.FormFile("video")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to parse form file", err)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid Content-Type", nil)
		return
	}
	if mediaType != "video/mp4" {
		respondWithError(w, http.StatusBadRequest, "Invalid file type", nil)
		return
	}

	tmpFile, err := os.CreateTemp("", "tubely-upload.mp4")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to create temp file", err)
		return
	}

	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err = io.Copy(tmpFile, file); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not write file to disk", err)
		return 
	}

	if _, err = tmpFile.Seek(0, io.SeekStart); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not reset file pointer", err)
		return
	}
	
	aspectRatio, err := getVideoAspectRatio(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error calculating video aspect ratio", err)
		return 
	}

	directory := ""
	switch aspectRatio {
		case "16:9":
			directory = "landscape"
		case "9:16":
			directory = "portrait"
		default:
			directory = "other"
	}

	processedFilePath, err := processVideoForFastStart(tmpFile.Name())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not proccess file", err)
		return
	}
	defer os.Remove(processedFilePath)

	processedFile, err := os.Open(processedFilePath)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not open proccessed file", err)
		return
	}
	defer processedFile.Close()

	key := getAssetPath(mediaType)
	key = directory + "/" + key
	_, err = cfg.s3Client.PutObject(r.Context(), &s3.PutObjectInput{
		Bucket: &cfg.s3Bucket,
		Key: &key,
		ContentType: &mediaType,
		Body: processedFile,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error uploading file to S3", err)
		return
	}

	// objectUrl := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", cfg.s3Bucket, cfg.s3Region, key)
	// objectUrl := fmt.Sprintf("%s,%s", cfg.s3Bucket, key)
	objectUrl := fmt.Sprintf("https://%s/%s", cfg.s3CfDistribution, key)
	video.VideoURL = &objectUrl

	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video file", err)
		return
	}

	// signedVideo, err := cfg.dbVideoToSignedVideo(video)
	// if err != nil {
	// 	respondWithError(w, http.StatusInternalServerError, "Couldn't generate presigned URL", err)
	// 	return
	// }

	respondWithJSON(w, http.StatusOK, video)
}

/*

func generatePresignedURL(s3Client *s3.Client, bucket, key string, expireTime time.Duration) (string, error) {
	s3PresignClient := s3.NewPresignClient(s3Client)
	
	params := &s3.GetObjectInput{
		Bucket: &bucket,
		Key: &key,
	}
	presignedHTTPRequest, err := s3PresignClient.PresignGetObject(context.TODO(), params, s3.WithPresignExpires(expireTime))
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %v", err)
	}

	return presignedHTTPRequest.URL, nil
}


func (cfg *apiConfig) dbVideoToSignedVideo(video database.Video) (database.Video, error) {
	if video.VideoURL == nil {
		return video, nil
	}
	splitStr := strings.Split(*video.VideoURL, ",")
	if len(splitStr) != 2 {
		return database.Video{}, errors.New("Invalid video URL, expected '<bucket>,<key>'")
	}
	bucket := splitStr[0]
	key := splitStr[1]

	signedURL, err := generatePresignedURL(cfg.s3Client, bucket, key, 5*time.Minute)
	if err != nil {
		return database.Video{}, err
	}
	video.VideoURL = &signedURL
	return video, nil
}

*/