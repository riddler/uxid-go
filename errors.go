package uxid

import "errors"

// The generation errors of the wire format, one per name in its closed set.
// Each error's text is exactly that name, the one the conformance corpus
// gives the refusal, so err.Error() on a value Generate returns is the
// corpus name; match them with errors.Is. Generate returns the first
// three and ErrRandWithFrom. ErrFromWithMonotonic and ErrRandWithMonotonic
// name the refusals of monotonic generation, which this package does not
// offer yet; nothing returns them today.
var (
	// ErrCompactTimeOutOfRange refuses a compact time whose first
	// character would be the deterministic marker: the low 40 bits of the
	// time are at or above 31 x 2^35.
	ErrCompactTimeOutOfRange = errors.New("compact_time_out_of_range")

	// ErrRandWidthMismatch refuses fixed random bytes (Options.Rand) whose
	// count is not the random width of the request.
	ErrRandWidthMismatch = errors.New("rand_width_mismatch")

	// ErrRandSizeTooLarge refuses an explicit random width
	// (Options.RandSize) above 10 bytes.
	ErrRandSizeTooLarge = errors.New("rand_size_too_large")

	// ErrFromWithMonotonic refuses a deterministic body asked for with
	// monotonic generation: a deterministic id has no random field to
	// step.
	ErrFromWithMonotonic = errors.New("from_with_monotonic")

	// ErrRandWithFrom refuses fixed random bytes asked for with a
	// deterministic body, which has no random field.
	ErrRandWithFrom = errors.New("rand_with_from")

	// ErrRandWithMonotonic refuses fixed random bytes asked for with
	// monotonic generation, which chooses the random field itself.
	ErrRandWithMonotonic = errors.New("rand_with_monotonic")
)

// The refusals of this package's own, which the wire format's closed set
// does not name. Each error's text is a snake_case name of the same shape.
var (
	// ErrTimeOutOfRange refuses a time (Options.Time, or the system clock)
	// before the Unix epoch or more than 2^48 - 1 milliseconds after it,
	// the latest a time field holds.
	ErrTimeOutOfRange = errors.New("time_out_of_range")

	// ErrNegativeRandSize refuses an explicit random width
	// (Options.RandSize) below 0.
	ErrNegativeRandSize = errors.New("negative_rand_size")

	// ErrUnknownSize refuses a Size outside the five constants, and is
	// what ParseSize returns for a string that is not a size's spelling.
	ErrUnknownSize = errors.New("unknown_size")

	// ErrUnknownCase refuses a Case outside the two constants.
	ErrUnknownCase = errors.New("unknown_case")
)
