package tuner

type Tuner interface {
	GetName() string
	// GetRecommendation returns a recommended replica count based on the current state and preferences.
	// Some tuners may not distinguish between scale up/down preferences, so the preference structure
	// is left to each tuner's implementation to interpret appropriately.
	GetRecommendation(keyForAutoscaler string,
		currentReplicas int32, desiredReplicas int32, preference interface{}) int32
	AcceptRecommendation(keyForAutoscaler string, currentReplicas int32, desiredReplicas int32)
}

func max(a, b int32) int32 {
	if a >= b {
		return a
	}
	return b
}

func min(a, b int32) int32 {
	if a <= b {
		return a
	}
	return b
}
