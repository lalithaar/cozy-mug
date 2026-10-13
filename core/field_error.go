package core

// FieldError is a validator's field-level message: which form field (by its
// input name) failed and why. It is defined here (rather than in the form
// package) so a model can return one without importing form or data -- the
// form package then maps these onto the Form's errors. (See form.Form.)
type FieldError struct {
	Field   string
	Message string
}
