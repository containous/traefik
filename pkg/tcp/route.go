package tcp

// Route holds matchers to match TCP routes.
type Route struct {
	// List of matchers that will be used to match the route.
	matchers []Matcher
	// Handler responsible for handling the route.
	handler Handler
}

// NewRoute returns a new Route.
func NewRoute(handler Handler) *Route {
	return &Route{handler: handler}
}

// Match checks the connection against all the matchers in the route, and returns if there is a full match.
func (r *Route) Match(conn WriteCloser) bool {
	// For each matcher, check if match, and return true if all are matched.
	for _, matcher := range r.matchers {
		if !matcher.Match(conn) {
			return false
		}
	}
	// All matchers matched
	return true
}

// AddMatcher adds a matcher to the route.
func (r *Route) AddMatcher(m Matcher) {
	r.matchers = append(r.matchers, m)
}
