package version

var Version = "0.0.0-dev"

func Title(appName string) string {
	return appName + " v" + Version
}
