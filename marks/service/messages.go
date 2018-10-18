package service

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/cube2222/usos-notifier/common/users"
	"github.com/cube2222/usos-notifier/marks"
	"github.com/cube2222/usos-notifier/marks/parser"
	"github.com/pkg/errors"
)

func (s *Service) SubscribeClass(ctx context.Context, userID users.UserID, params map[string]string) (string, error) {
	httpCli := &http.Client{}
	classID := params["class_id"]

	user, err := s.users.Get(ctx, userID)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get user")
	}

	// Check if the user isn't already subscribed to this class
	for _, class := range user.ObservedClasses {
		if class.ID == classID {
			return "You're already subscribed to this class.", nil
		}
	}

	session, err := s.getSession(ctx, userID)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get session")
	}

	// Check if the class is already known to us
	var found *marks.ClassHeader
	for _, class := range user.AvailableClasses {
		if class.ID == classID {
			found = &class
			break
		}
	}

	// Check if this is a class that is available, but we don't know about it yet
	if found == nil {
		classes, err := getClasses(ctx, httpCli, session)
		if err != nil {
			return "", errors.Wrap(err, "couldn't get classes")
		}
		user.AvailableClasses = make([]marks.ClassHeader, 0)
		for id, class := range classes {
			user.AvailableClasses = append(user.AvailableClasses, marks.ClassHeader{
				ID:   id,
				Name: class.Name,
			})
		}
	}

	// Check the new - updated - list of available classes
	for _, class := range user.AvailableClasses {
		if class.ID == classID {
			found = &class
			break
		}
	}

	// If we haven't found it yet, then it doesn't exist for sure.
	if found == nil {
		return "No class with this ID is available.", nil
	}

	scores, err := getScoresForClass(ctx, httpCli, session, found.ID)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get scores for class")
	}

	user.ObservedClasses = append(user.ObservedClasses, *found)

	user.Classes = append(user.Classes, parser.MakeClassWithScores(found.ID, found.Name, scores))
	sort.Slice(user.Classes, func(i, j int) bool {
		return user.Classes[i].ID < user.Classes[j].ID
	})

	err = s.users.Set(ctx, userID, user)
	if err != nil {
		return "", errors.Wrap(err, "couldn't save user")
	}

	return fmt.Sprintf("Successfully subscribed to %s", found.Name), nil
}

func (s *Service) UnsubscribeClass(ctx context.Context, userID users.UserID, params map[string]string) (string, error) {
	classID := params["class_id"]

	user, err := s.users.Get(ctx, userID)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get user")
	}

	foundIndex := -1
	for i := range user.ObservedClasses {
		if user.ObservedClasses[i].ID == classID {
			foundIndex = i
			break
		}
	}

	if foundIndex == -1 {
		return "It seems like you've not been subscribed to this class.", nil
	}

	if foundIndex == len(user.ObservedClasses)-1 {
		user.ObservedClasses = user.ObservedClasses[:foundIndex]
	} else {
		user.ObservedClasses = append(user.ObservedClasses[:foundIndex], user.ObservedClasses[foundIndex+1:]...)
	}

	for i := range user.Classes {
		if user.Classes[i].ID == classID {
			if i == len(user.Classes)-1 {
				user.Classes = user.Classes[:i]
			} else {
				user.Classes = append(user.Classes[:i], user.Classes[i+1:]...)
			}
			break
		}
	}

	err = s.users.Set(ctx, userID, user)
	if err != nil {
		return "", errors.Wrap(err, "couldn't save user")
	}

	return "Successfully unsubscribed.", nil
}

func (s *Service) ListClasses(ctx context.Context, userID users.UserID, params map[string]string) (string, error) {
	user, err := s.users.Get(ctx, userID)
	if err != nil {
		return "", errors.Wrap(err, "couldn't get user")
	}

	observedSet := map[string]struct{}{}
	for _, class := range user.ObservedClasses {
		observedSet[class.ID] = struct{}{}
	}

	lines := make([]string, len(user.AvailableClasses)+1)
	lines[0] = "These are your classes (* for subscribed):"
	for i, class := range user.AvailableClasses {
		if _, ok := observedSet[class.ID]; ok {
			lines[i+1] = fmt.Sprintf("* %v: %v", class.ID, class.Name)
		} else {
			lines[i+1] = fmt.Sprintf("%v: %v", class.ID, class.Name)
		}
	}

	return strings.Join(lines, "\n"), nil
}
