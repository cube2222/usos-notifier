package service

import (
	"time"

	"github.com/beefsack/go-rate"

	"github.com/cube2222/usos-notifier/notifier"
)

type RateLimitReason int

const (
	ReasonUser RateLimitReason = iota
	ReasonGeneral
)

func (r RateLimitReason) String() string {
	switch r {
	case ReasonUser:
		return "user limit"
	case ReasonGeneral:
		return "general limit"
	default:
		return "unknown limit"
	}
}

type RateLimit struct {
	Reason   RateLimitReason
	TimeLeft time.Duration
}

func NewRateLimit(reason RateLimitReason, left time.Duration) *RateLimit {
	return &RateLimit{
		Reason:   reason,
		TimeLeft: left,
	}
}

type MessengerRateLimiter struct {
	PerHourUser    int
	PerHourGeneral int
	userLimiters   map[notifier.MessengerID]*rate.RateLimiter
	generalLimiter *rate.RateLimiter
}

func NewMessengerRateLimiter(userPerHour, generalPerHour int) *MessengerRateLimiter {
	return &MessengerRateLimiter{
		PerHourUser:    userPerHour,
		PerHourGeneral: generalPerHour,
		userLimiters:   make(map[notifier.MessengerID]*rate.RateLimiter),
		generalLimiter: rate.New(generalPerHour, time.Hour),
	}
}

func (rl *MessengerRateLimiter) LimitMessengerUser(userID notifier.MessengerID) (limit *RateLimit, limited bool) {
	userLimiter, ok := rl.userLimiters[userID]
	if !ok {
		userLimiter = rate.New(rl.PerHourUser, time.Hour)
		rl.userLimiters[userID] = userLimiter
	}

	ok, left := userLimiter.Try()
	if !ok {
		return NewRateLimit(ReasonUser, left), true
	}
	ok, left = rl.generalLimiter.Try()
	if !ok {
		return NewRateLimit(ReasonGeneral, left), true
	}
	return nil, false
}
