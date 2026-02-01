package isp

import (
	"log/slog"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	MGR_CTL_PATH_DEFAULT = "/usr/local/mgr5/sbin/mgrctl"
	MGR_WEBDOMAIN_REGEX  = `id=(?P<id>\d+)\s+name=(?P<name>[\w\.\-]+)\s+owner=(?P<owner>\w+)\s+docroot=(?P<docroot>[\w/\.\-]+)\s+(?:secure=(?P<secure>\w+)\s+)?php=(?P<php>.*?)\s+php_mode=(?P<php_mode>\w+)\s+php_version=(?P<php_version>[\d\.]+ \([^)]+\))\s+handler=(?P<handler>.*?)\s+active=(?P<active>\w+)\s+analyzer=(?P<analyzer>\w+)\s+ipaddr=(?P<ipaddr>[\d\.]+)\s+webscript_status=(?P<webscript_status>\w*)\s+database=(?P<database>[\w_\.\-]+)\s+(?P<ssl_status>[\w_]+)=?`
)

var execCommand = exec.Command

func GetWebDomain(mgrctlPath string) ([]*WebDomain, error) {
	cmd := execCommand(mgrctlPath, "-m", "ispmgr", "webdomain")

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// Компилируем регулярку один раз
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
			slog.Warn("строка не соответствует регулярному выражению", "line", line)
			continue
		}

		domain := &WebDomain{
			Port: "80",
		}

		for i, name := range names {
			switch name {
			case "id":
				if err := setIntVal(&domain.Id, match[i]); err != nil {
					slog.Warn("ошибка парсинга ID", "line", line, "error", err)
					continue
				}
			case "name":
				domain.Name = match[i]
			case "owner":
				domain.Owner = match[i]
			case "docroot":
				domain.Docroot = match[i]
			case "active":
				domain.Active = match[i] == "on"
			case "ipaddr":
				domain.IPAddr = match[i]
			case "ssl_status":
				if !strings.EqualFold(match[i], "ssl_not_used") {
					domain.Port = "443"
				}
			}
		}

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
