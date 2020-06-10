package env

type Environment struct {
	Scope     string
	global    KeyValues
	overrides KeyValues
}
