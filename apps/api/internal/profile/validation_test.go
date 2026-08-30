package profile

import (
	"testing"
)

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }

func TestValidateUpdateRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     UpdateRequest
		wantErr bool
	}{
		{
			name: "valid",
			req: UpdateRequest{
				FirstName:       strPtr("Jordan"),
				LastName:        strPtr("Smith"),
				GraduationYear:  intPtr(2027),
				WorkArrangement: strPtr("remote"),
				ExperienceLevel: strPtr("intern"),
				GithubURL:       strPtr("https://github.com/jordan"),
			},
			wantErr: false,
		},
		{
			name: "name too long",
			req: UpdateRequest{
				FirstName: strPtr(string(make([]byte, 101))),
			},
			wantErr: true,
		},
		{
			name: "invalid graduation year",
			req: UpdateRequest{
				GraduationYear: intPtr(2010),
			},
			wantErr: true,
		},
		{
			name: "invalid work arrangement",
			req: UpdateRequest{
				WorkArrangement: strPtr("onsite"),
			},
			wantErr: true,
		},
		{
			name: "invalid experience level",
			req: UpdateRequest{
				ExperienceLevel: strPtr("expert"),
			},
			wantErr: true,
		},
		{
			name: "invalid URL",
			req: UpdateRequest{
				GithubURL: strPtr("not-a-url"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUpdateRequest(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUniversityOrDefault(t *testing.T) {
	if got := universityOrDefault(nil); got != defaultUniversity {
		t.Errorf("expected default university, got %s", got)
	}
	empty := ""
	if got := universityOrDefault(&empty); got != defaultUniversity {
		t.Errorf("expected default university for empty string, got %s", got)
	}
	custom := "Howard University"
	if got := universityOrDefault(&custom); got != custom {
		t.Errorf("expected custom university, got %s", got)
	}
}

func TestNormalizeStringSlice(t *testing.T) {
	if got := normalizeStringSlice(nil); len(got) != 0 {
		t.Error("expected empty slice for nil input")
	}
}
