package version

import "runtime/debug"

const Version = "0.9.1"

func Info() map[string]interface{} {
	info := map[string]interface{}{"version": Version}
	if build, ok := debug.ReadBuildInfo(); ok {
		info["go_version"] = build.GoVersion
		for _, setting := range build.Settings {
			switch setting.Key {
			case "vcs.revision":
				info["revision"] = setting.Value
			case "vcs.modified":
				info["modified"] = setting.Value == "true"
			}
		}
	}
	return info
}
