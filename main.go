package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type SaveFileRequest struct {
	FileBody string `json:"FileBody"` // Base64 encoded file content
	FileName string `json:"FileName"`
}

type SaveFileResponse struct {
	Success bool   `json:"Success"`
	Message string `json:"Message"`
}

type DeleteFileRequest struct {
	FileName string `json:"FileName"`
}

type ExtractFileRequest struct {
	FileName string `json:"FileName"`
}

type Config struct {
	StorageDir string `json:"storage_dir"`
}

var storageDir = "" //= "C:/Users/Domiurg/Documents/GO/FileServerStorage/Storage" // Директория для хранения файлов

func handlerSaveFileToStorage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendErrorResponse(w, "Error reading request body")
		return
	}
	defer r.Body.Close()

	var requestData SaveFileRequest
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		sendErrorResponse(w, "Error parsing JSON")
		return
	}

	fileData, err := base64.StdEncoding.DecodeString(requestData.FileBody)
	if err != nil {
		log.Printf("Base64 decode error: %v, input: %s", err, requestData.FileBody)
		sendErrorResponse(w, "Error decoding base64")
		return
	}

	err = os.MkdirAll(storageDir, os.ModePerm)
	if err != nil {
		log.Printf("Error creating directory: %v", err)
		sendErrorResponse(w, "Error creating storage directory")
		return
	}

	filePath := filepath.Join(storageDir, requestData.FileName)

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		sendErrorResponse(w, "Error opening file")
		return
	}
	defer file.Close()

	_, err = io.Copy(file, bytes.NewReader(fileData))
	if err != nil {
		log.Printf("Error writing to file: %v", err)
		sendErrorResponse(w, "Error writing file")
		return
	}

	err = file.Sync()
	if err != nil {
		log.Printf("Error syncing file: %v", err)
		sendErrorResponse(w, "Error syncing file")
		return
	}
	//ОЧЕНЬ ВАЖНО!!!
	err = file.Close()
	if err != nil {
		log.Printf("Error closing file: %v", err)
		sendErrorResponse(w, "Error closing file")
		return
	}

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		log.Printf("Error getting file info: %v", err)
		sendErrorResponse(w, "Error getting file info")
		return
	}
	log.Printf("File size after writing: %d bytes", fileInfo.Size())

	sendSuccessResponse(w)
}

func handlerDeleteFileInStorage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendErrorResponse(w, "Error reading request body")
		return
	}
	defer r.Body.Close()

	var requestData DeleteFileRequest
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		sendErrorResponse(w, "Error parsing JSON")
		return
	}

	filePath := filepath.Join(storageDir, requestData.FileName)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		sendErrorResponse(w, "File not found")
		return
	}

	err = os.Remove(filePath)
	if err != nil {
		sendErrorResponse(w, "Error deleting file")
		return
	}

	sendSuccessResponse(w)
}

func handlerExtractFromStorage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		sendErrorResponse(w, "Error reading request body")
		return
	}
	defer r.Body.Close()

	var requestData ExtractFileRequest
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		sendErrorResponse(w, "Error parsing JSON")
		return
	}

	filePath := filepath.Join(storageDir, requestData.FileName)

	// Логируем путь к файлу для отладки
	log.Printf("Attempting to open file: %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening file: %v", err) // Логируем ошибку открытия файла
		if os.IsNotExist(err) {
			sendErrorResponse(w, fmt.Sprintf("File not found: %s", requestData.FileName)) // Более информативное сообщение
		} else {
			sendErrorResponse(w, fmt.Sprintf("Error opening file: %v", err))
		}
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Error reading file content: %v", err) // Логируем ошибку чтения
		sendErrorResponse(w, "Error reading file content")
		return
	}

	base64Data := base64.StdEncoding.EncodeToString(fileData)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		FileBody string `json:"FileBody"`
		Success  bool   `json:"Success"`
		Message  string `json:"Message"`
	}{
		FileBody: base64Data,
		Success:  true,
		Message:  "",
	})
}

func sendErrorResponse(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest) // Устанавливаем код ошибки
	json.NewEncoder(w).Encode(SaveFileResponse{Success: false, Message: message})
}

func sendSuccessResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(SaveFileResponse{Success: true, Message: ""})
}

func loadConfig(filename string) (Config, error) {
	var config Config
	file, err := os.Open(filename)
	if err != nil {
		return Config{}, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

func main() {

	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	storageDir = config.StorageDir // Теперь storageDir берется из конфига
	fmt.Printf("Storage directory: %s\n", storageDir)

	http.HandleFunc("/SaveFileToStorage", handlerSaveFileToStorage)
	http.HandleFunc("/DeleteFileInStorage", handlerDeleteFileInStorage)
	http.HandleFunc("/ExtractFromStorage", handlerExtractFromStorage)

	fmt.Println("Server listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
