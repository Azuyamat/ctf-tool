package config

type Key struct {
	Name        string
	Description string
	Get         func(*Config) string
	Set         func(*Config, string) error
}
