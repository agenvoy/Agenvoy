package summary

func IsValid(dic map[string]any) bool {
	newKeys := []string{"key_decisions", "past_discussions", "current_discussion"}
	return match(dic, newKeys) >= 2
}

func match(dic map[string]any, keys []string) int {
	num := 0
	for _, key := range keys {
		if _, ok := dic[key]; ok {
			num++
		}
	}
	return num
}
