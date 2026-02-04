package isp

type WebDomain struct {
	Id      int
	Name    string
	Owner   string
	Docroot string
	Active  bool
	IPAddr  string
	Port    string
	Sites   []string
}
