package isp

type WebDomain struct {
	Id      int
	Name    string
	Owner   string
	Docroot string
	IPAddr  string
	Port    string
	Sites   []string
}
