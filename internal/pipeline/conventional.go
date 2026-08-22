package pipeline

import "fmt"

func conventionalType(bump string) (typ string, breaking bool) {
	switch bump {
	case "patch":
		return "fix", false
	case "minor":
		return "feat", false
	case "major":
		return "feat", true
	default:
		return "", false
	}
}

func conventionalSubject(typ string, breaking bool, title, id string) string {
	bang := ""
	if breaking {
		bang = "!"
	}
	return fmt.Sprintf("%s%s: %s (%s)", typ, bang, title, id)
}
