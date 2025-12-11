// Copyright (C) MongoDB, Inc. 2017-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package bsoncore

// Truncate truncates a string to maxLen bytes, ensuring UTF-8 validity
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	
	// Ensure we don't cut in the middle of a UTF-8 character
	for maxLen > 0 && maxLen < len(s) {
		if (s[maxLen] & 0xC0) != 0x80 {
			break
		}
		maxLen--
	}
	
	return s[:maxLen]
}
