package shiftnotifier

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

func (service *service) diffToMessage(diffs *shiftDiffs) (string, string) {
	msg := strings.Builder{}
	msgHTML := strings.Builder{}

	defaultTZ, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		defaultTZ = time.Local
	}

	timeStr := diffs.ReferenceTime.
		Add(service.config.NotifyBeforeShiftStart).
		In(defaultTZ).
		Format("Mon, 15:04")

	// Title.
	msg.WriteString("TROLL CHANGES FOR ")
	msg.WriteString(timeStr)
	msg.WriteString("\n\n")

	msgHTML.WriteString("<h1>Troll Changes for ")
	msgHTML.WriteString(timeStr)
	msgHTML.WriteString("</h1><br>\n")

	// Sort by mapkey to have deterministic order.
	locations := make([]string, 0, len(diffs.DiffsInLocations))
	for k := range diffs.DiffsInLocations {
		locations = append(locations, k)
	}
	sort.Strings(locations)

	for _, loc := range locations {
		// Location.
		msg.WriteString("📍 ")
		msg.WriteString(loc)
		msg.WriteString("\n")

		msgHTML.WriteString("📍 <b>")
		msgHTML.WriteString(loc)
		msgHTML.WriteString("</b><br>\n")

		// Troll lists.
		msg.WriteString("Arriving Trolls 🔜:\n")
		msgHTML.WriteString("Arriving Trolls 🔜:<br>\n")
		usersToList(diffs.DiffsInLocations[loc].UsersArriving, &msg, &msgHTML)

		msg.WriteString("Staying Trolls 🔄:\n")
		msgHTML.WriteString("Staying Trolls 🔄:<br>\n")
		usersToList(diffs.DiffsInLocations[loc].UsersWorking, &msg, &msgHTML)

		msg.WriteString("Leaving Trolls 🔚:\n")
		msgHTML.WriteString("Leaving Trolls 🔚:<br>\n")
		usersToList(diffs.DiffsInLocations[loc].UsersLeaving, &msg, &msgHTML)

		// Summary.
		msg.WriteString("\nExpecting ")
		msg.WriteString(strconv.Itoa(int(diffs.DiffsInLocations[loc].ExpectedUsers)))
		msg.WriteString(" trolls total\n")
		msgHTML.WriteString("<br>\nExpecting ")
		msgHTML.WriteString(strconv.Itoa(int(diffs.DiffsInLocations[loc].ExpectedUsers)))
		msgHTML.WriteString(" trolls total<br>\n")

		if len(diffs.DiffsInLocations[loc].OpenUsers) > 0 {
			msg.WriteString("🚨 Open positions:\n")
			for shiftType, amount := range diffs.DiffsInLocations[loc].OpenUsers {
				msg.WriteString("- ")
				msg.WriteString(strconv.Itoa(int(amount)))
				msg.WriteString("x ")
				msg.WriteString(shiftType)
				msg.WriteString("\n")
			}

			msgHTML.WriteString("🚨 Open positions:<br>\n")
			for shiftType, amount := range diffs.DiffsInLocations[loc].OpenUsers {
				msgHTML.WriteString("- ")
				msgHTML.WriteString(strconv.Itoa(int(amount)))
				msgHTML.WriteString("x ")
				msgHTML.WriteString(shiftType)
				msgHTML.WriteString("<br>\n")
			}
		}

		msg.WriteString("\n")
		msgHTML.WriteString("<br>\n")
	}

	return msg.String(), msgHTML.String()
}

func usersToList(users []shiftUser, msg *strings.Builder, msgHTML *strings.Builder) {
	if len(users) == 0 {
		msg.WriteString("  _none_\n")
		msgHTML.WriteString("&nbsp;&nbsp;<i>none</i><br>\n")
		return
	}

	for _, user := range users {
		msg.WriteString("  - ")
		msg.WriteString(user.Nickname)
		msg.WriteString(" (")
		msg.WriteString(user.ShiftName)
		msg.WriteString(shiftNameToEmoji(user.ShiftName))
		msg.WriteString(")\n")

		msgHTML.WriteString("&nbsp;&nbsp;- ")
		msgHTML.WriteString(user.Nickname)
		msgHTML.WriteString(" <i>(")
		msgHTML.WriteString(user.ShiftName)
		msgHTML.WriteString(shiftNameToEmoji(user.ShiftName))
		msgHTML.WriteString(")</i><br>\n")
	}
}

func shiftNameToEmoji(shiftName string) string {
	shiftName = strings.TrimSpace(strings.ToLower(shiftName))
	switch {
	case strings.Contains(shiftName, "orga"):
		return " 👑"
	case strings.Contains(shiftName, "tschunk"):
		return " 🥃"
	case strings.Contains(shiftName, "kaffee"):
		return " 🍫"
	case strings.Contains(shiftName, "runner"):
		return " 🏃‍♀️"
	case strings.Contains(shiftName, "bottle"):
		return " ♻️"
	case strings.Contains(shiftName, "bar-theke"):
		return " 💶"
	default:
		return ""
	}
}
