package git

import "regexp"

// notFoundPatterns matches error text that git and common hosting providers
// (GitHub, GitLab, Bitbucket, Azure DevOps) emit when a clone target does
// not exist anymore, as opposed to transient network or authentication
// failures. Git itself does not expose a distinct exit code for "not
// found", so this is a best-effort heuristic over combined command output.
var notFoundPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)repository not found`),
	regexp.MustCompile(`(?i)could not be found`),
	regexp.MustCompile(`(?i)requested url returned error: 404`),
	regexp.MustCompile(`(?i)does not exist or you do not have permission`),
	// Generic git transport message for a remote that doesn't resolve to a
	// repository at all (host reachable, nothing there). This is what plain
	// git servers, file:// remotes, and some SSH setups report instead of a
	// provider-specific "not found" line. Deliberately narrower than the
	// "Could not read from remote repository / access rights" text that
	// follows it, since that combination is also the standard message for a
	// bad or missing SSH key (a false positive we want to avoid pruning on).
	regexp.MustCompile(`(?i)does not appear to be a git repository`),
}

// IsNotFoundError reports whether err looks like a clone failure caused by a
// remote repository that no longer exists (deleted, renamed, or never
// existed), rather than a network, auth, or local filesystem failure.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	for _, re := range notFoundPatterns {
		if re.MatchString(msg) {
			return true
		}
	}
	return false
}
