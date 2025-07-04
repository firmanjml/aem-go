package errors

import "fmt"

type AEMError struct {
	Type    string
	Message string
	Cause   error
}

func (e *AEMError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func NewDownloadError(message string, cause error) *AEMError {
	return &AEMError{Type: "DOWNLOAD_ERROR", Message: message, Cause: cause}
}

func NewExtractionError(message string, cause error) *AEMError {
	return &AEMError{Type: "EXTRACTION_ERROR", Message: message, Cause: cause}
}

func NewFileSystemError(message string, cause error) *AEMError {
	return &AEMError{Type: "FILESYSTEM_ERROR", Message: message, Cause: cause}
}

func FileWriteSystemError(message string, cause error) *AEMError {
	return &AEMError{Type: "FILE_WRITE_ERROR", Message: message, Cause: cause}
}

func NewAPIError(message string, cause error) *AEMError {
	return &AEMError{Type: "API_ERROR", Message: message, Cause: cause}
}

func NewValidationError(message string) *AEMError {
	return &AEMError{Type: "VALIDATION_ERROR", Message: message}
}

func UninstallError(message string, cause error) *AEMError {
	return &AEMError{Type: "UNINSTALL_ERROR", Message: message, Cause: cause}
}
