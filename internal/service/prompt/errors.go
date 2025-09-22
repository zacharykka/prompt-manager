package prompt

import "errors"

var (
    ErrNameRequired        = errors.New("prompt name required")
    ErrBodyRequired        = errors.New("prompt body required")
    ErrPromptNotFound      = errors.New("prompt not found")
    ErrVersionNotFound     = errors.New("prompt version not found")
    ErrPromptAlreadyExists = errors.New("prompt already exists")
)
