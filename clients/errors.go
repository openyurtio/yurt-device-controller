package clients

// import "errors"
import "strings"

type NotFoundError struct{}

func (e *NotFoundError) Error() string { return "Item not found" }

func IsNotFoundErr(err error) bool {
	return err.Error() == "Item not found" || strings.HasPrefix(err.Error(), "no item found")
	// return errors.Is(err, &NotFoundError{})
}
