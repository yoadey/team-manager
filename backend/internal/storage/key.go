package storage

// TeamPhotoKey returns the object key for teamID's photo.
func TeamPhotoKey(teamID string) string { return "teams/" + teamID + "/photo" }

// TeamLogoKey returns the object key for teamID's logo.
func TeamLogoKey(teamID string) string { return "teams/" + teamID + "/logo" }

// UserPhotoKey returns the object key for userID's profile photo.
func UserPhotoKey(userID string) string { return "users/" + userID + "/photo" }
