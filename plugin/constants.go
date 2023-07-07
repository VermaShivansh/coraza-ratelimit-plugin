package plugin

type ConfigTestCase struct {
	ID     int64
	Config string
	// true means pass, false means fail
	Expected bool
}

var ConfigTestCases = []ConfigTestCase{
	{ID: 1,
		Config:   "events=200&window=1", // zone is required should fail
		Expected: false},
	{ID: 2,
		Config:   "zone=fixed_value&events=200&window=1", // not using a macro should work as well...in this case ratelimit will not be divided on bases of headers, matching variables and so on... it will be constant for all
		Expected: true},
	{ID: 3,
		Config:   "zone=%{REQUEST_HEADERS.authority&window=1", // in case of wrong macro format it, error is returned from macro parser end, therefore should fail
		Expected: false},
	{ID: 4,
		Config:   "zone=%{REQUEST_HEADERS.authority}&window=1", // events is required
		Expected: false},
	{ID: 5,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=0&window=1", // events can be 0 should pass; helpful in cases when we want to completely block request
		Expected: true},
	{ID: 6,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=abc&window=1", // events cannot be string
		Expected: false},
	{ID: 7,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100", // window is required
		Expected: false},
	{ID: 8,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100&window=0", // window cannot be 0
		Expected: false},
	{ID: 9,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100&window=ab", // window cannot be a string
		Expected: false},
	{ID: 10,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100&window=2", // only the required fields are given it should pass
		Expected: true},
	{ID: 11,
		Config:   "zone=%{REQUEST_HEADERS.host}&&events=100&window=2", // irregular use of & should fail; & in middle somwhere
		Expected: false},
	{ID: 12,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100&window=2&", // irregular use of & should fail; & in the end
		Expected: false},
	{ID: 13,
		Config:   "&zone=%{REQUEST_HEADERS.host}&events=100&window=2", // irregular use of & should fail; & in beginning
		Expected: false},
	{ID: 14,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100&window=2&interval=0&action=drop&status=429", // interval cannot be 0
		Expected: false},
	{ID: 15,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100&window=2&interval=ab&action=drop&status=429", // interval cannot be string
		Expected: false},
	{ID: 16,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100&window=2&interval=2&action=dro&status=429", // action must be one of 'drop' 'deny' or 'redirect'
		Expected: false},
	{ID: 17,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100&window=2&interval=2&action=drop&status=429", // action 'drop' should be allowed
		Expected: true},
	{ID: 18,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100&window=2&interval=2&action=deny&status=403", // action 'deny' should be allowed
		Expected: true},
	{ID: 19,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100&window=2&interval=2&action=redirect&status=301", // action 'redirect' should be allowed
		Expected: true},
	{ID: 20,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100&window=2&interval=2&action=drop&status=-1", // status must be from 0 to 500
		Expected: false},
	{ID: 21,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100&window=2&interval=2&action=drop&status=501", // status must be from 0 to 500
		Expected: false},
	{ID: 22,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100&window=2&interval=2&action=drop&status=0", // boundary condition for 0 should be allowed
		Expected: true},
	{ID: 23,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100&window=2&interval=2&action=drop&status=500", // boundary condiiton for 500 should be allowed
		Expected: true},
	{ID: 24,
		Config:   "zone=%{REQUEST_HEADERS.host}&events=100&window=2&interval=2&action=drop&status=abc", // status cannot be string
		Expected: false},
}
