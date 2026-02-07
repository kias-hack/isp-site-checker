package isp

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var readDir = os.ReadDir

const (
	MGR_CTL_PATH_DEFAULT = "/usr/local/mgr5/sbin/mgrctl"
	MGR_WEBDOMAIN_REGEX  = `id=(?P<id>\d+)\s+name=(?P<name>[\w\.\-]+)\s+owner=(?P<owner>\w+)\s+docroot=(?P<docroot>[\w/\.\-]+)\s+(?:secure=(?P<secure>\w+)\s+)?php=(?P<php>.*?)\s+php_mode=(?P<php_mode>\w+)\s+php_version=(?P<php_version>[\d\.]+ \([^)]+\))\s+handler=(?P<handler>.*?)\s+active=(?P<active>\w+)\s+analyzer=(?P<analyzer>\w+)\s+ipaddr=(?P<ipaddr>[\d\.]+)\s+webscript_status=(?P<webscript_status>\w*)\s+database=(?P<database>[\w_\.\-]+)\s+(?P<ssl_status>[\w_]+)=?`
)

type GetWebDomainsFunc func() ([]*WebDomain, error)

var execCommand = exec.Command

func GetWebDomains(mgrctlPath string) ([]*WebDomain, error) {
	cmd := execCommand(mgrctlPath, "-m", "ispmgr", "webdomain")

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Compile regex once
	re, err := regexp.Compile(MGR_WEBDOMAIN_REGEX)
	if err != nil {
		return nil, err
	}

	domainRows := strings.Split(string(output), "\n")
	result := []*WebDomain{}
	names := re.SubexpNames()

	for _, line := range domainRows {
		if len(line) == 0 {
			continue
		}

		match := re.FindStringSubmatch(line)
		if match == nil {
			slog.Warn("line does not match regex", "line", line)
			continue
		}

		domain := &WebDomain{
			Port: "80",
		}

		skip := false

		for i, name := range names {
			switch name {
			case "id":
				if err := setIntVal(&domain.Id, match[i]); err != nil {
					slog.Warn("failed to parse ID", "line", line, "error", err)
					continue
				}
			case "name":
				domain.Name = match[i]
			case "owner":
				domain.Owner = match[i]
			case "docroot":
				domain.Docroot = match[i]
			case "active":
				skip = match[i] != "on"
			case "ipaddr":
				domain.IPAddr = match[i]
			case "ssl_status":
				if !strings.EqualFold(match[i], "ssl_not_used") {
					domain.Port = "443"
				}
			}
		}

		if skip {
			continue
		}

		domain.Sites = findSubdomain(domain.Owner, domain.Name)

		result = append(result, domain)
	}

	return result, nil
}

func setIntVal(target *int, value string) error {
	val, err := strconv.Atoi(value)
	if err != nil {
		return err
	}

	*target = val

	return nil
}

func findSubdomain(owner string, domain string) []string {
	path := fmt.Sprintf("/var/www/%s/data/www/", owner)

	result := []string{domain}

	domain = "." + domain

	entries, err := readDir(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		panic(fmt.Errorf("unknown error while read dir %s: %w", path, err))
	}

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == domain {
			continue
		}

		if strings.HasSuffix(entry.Name(), domain) {
			result = append(result, entry.Name())
		}
	}

	return result
}
