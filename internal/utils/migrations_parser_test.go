package utils

import "testing"

func TestParseSQL(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []MigrationProto
		wantErr bool
	}{
		{
			name: "single migration",
			content: `-- :user1:ns1:comment1
CREATE TABLE users (id INT);`,
			want: []MigrationProto{
				{
					ShouldSkip: false,
					User:       "user1",
					Namespace:  "ns1",
					Comment:    "comment1",
					DDL:        "CREATE TABLE users (id INT);",
					Signature:  4193559969700021025,
				},
			},
			wantErr: false,
		},
		{
			name: "multiple migrations",
			content: `-- :user1:ns1:comment1
CREATE TABLE users (id INT);

-- skip:user2:ns2:comment2
ALTER TABLE users ADD COLUMN name TEXT;`,
			want: []MigrationProto{
				{
					ShouldSkip: false,
					User:       "user1",
					Namespace:  "ns1",
					Comment:    "comment1",
					DDL:        "CREATE TABLE users (id INT);",
					Signature:  4193559969700021025,
				},
				{
					ShouldSkip: true,
					User:       "user2",
					Namespace:  "ns2",
					Comment:    "comment2",
					DDL:        "ALTER TABLE users ADD COLUMN name TEXT;",
					Signature:  9442060313613740461,
				},
			},
			wantErr: false,
		},
		{
			name: "multi-line SQL",
			content: `-- :user1:ns1:comment1
CREATE TABLE users (
    id INT,
    name TEXT
);`,
			want: []MigrationProto{
				{
					ShouldSkip: false,
					User:       "user1",
					Namespace:  "ns1",
					Comment:    "comment1",
					DDL:        "CREATE TABLE users (\n    id INT,\n    name TEXT\n);",
					Signature:  4193559969700021025,
				},
			},
			wantErr: false,
		},
		{
			name:    "empty content",
			content: "",
			want:    []MigrationProto{},
			wantErr: false,
		},
		{
			name:    "only whitespace",
			content: "   \n\t\n  ",
			want:    []MigrationProto{},
			wantErr: false,
		},
		{
			name: "invalid header",
			content: `-- invalid
CREATE TABLE users;`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "missing SQL after header",
			content: "-- :user1:ns1:comment1",
			want:    []MigrationProto{},
			wantErr: false,
		},
		{
			name:    "SQL without header",
			content: "CREATE TABLE users;",
			want:    []MigrationProto{},
			wantErr: true,
		},
		{
			name: "header without SQL followed by another header",
			content: `-- :user1:ns1:comment1
-- :user2:ns2:comment2
CREATE TABLE users;`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "header followed by only whitespace",
			content: `-- :user1:ns1:comment1
    
    
-- :user2:ns2:comment2
CREATE TABLE users;`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSQL(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if len(got) != len(tt.want) {
				t.Errorf("ParseSQL() got %v migrations, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i].ShouldSkip != tt.want[i].ShouldSkip {
					t.Errorf("Migration[%d] ShouldSkip = %v, want %v", i, got[i].ShouldSkip, tt.want[i].ShouldSkip)
				}
				if got[i].User != tt.want[i].User {
					t.Errorf("Migration[%d] User = %v, want %v", i, got[i].User, tt.want[i].User)
				}
				if got[i].Namespace != tt.want[i].Namespace {
					t.Errorf("Migration[%d] Namespace = %v, want %v", i, got[i].Namespace, tt.want[i].Namespace)
				}
				if got[i].Comment != tt.want[i].Comment {
					t.Errorf("Migration[%d] Comment = %v, want %v", i, got[i].Comment, tt.want[i].Comment)
				}
				if got[i].DDL != tt.want[i].DDL {
					t.Errorf("Migration[%d] DDL = %v, want %v", i, got[i].DDL, tt.want[i].DDL)
				}
				if got[i].Signature != tt.want[i].Signature {
					t.Errorf("Migration[%d] Signature = %v, want %v", i, got[i].Signature, tt.want[i].Signature)
				}
			}
		})
	}
}

func TestGenerateSignature(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   uint64
	}{
		{
			name:   "simple header",
			header: "-- :user:ns:comment",
			want:   1374584940602396620, // pre-computed value
		},
		{
			name:   "header with skip",
			header: "-- skip:user:ns:comment",
			want:   15469498398215482039, // pre-computed value
		},
		{
			name:   "empty string",
			header: "",
			want:   14695981039346656037, // pre-computed value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateSignature(tt.header)
			if got != tt.want {
				t.Errorf("generateSignature() = %v, want %v", got, tt.want)
			}

			// Verify signature is consistent
			second := generateSignature(tt.header)
			if got != second {
				t.Errorf("generateSignature() not consistent: first = %v, second = %v", got, second)
			}
		})
	}
}

func TestParseMigrationHeader(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *MigrationProto
		wantErr bool
	}{
		{
			name:  "header without skip",
			input: "-- :username:prod:create users table",
			want: &MigrationProto{
				ShouldSkip: false,
				User:       "username",
				Namespace:  "prod",
				Comment:    "create users table",
			},
			wantErr: false,
		},
		{
			name:  "header with skip",
			input: "-- skip:janedoe:staging:alter users table",
			want: &MigrationProto{
				ShouldSkip: true,
				User:       "janedoe",
				Namespace:  "staging",
				Comment:    "alter users table",
			},
			wantErr: false,
		},
		{
			name:  "header with extra spaces",
			input: "--   skip  :  admin  :  test  :  add index  ",
			want: &MigrationProto{
				ShouldSkip: true,
				User:       "admin",
				Namespace:  "test",
				Comment:    "add index",
			},
			wantErr: false,
		},
		{
			name:    "invalid header - too few parts",
			input:   "-- user:namespace",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid header - empty string",
			input:   "",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid header - only comment markers",
			input:   "--",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "header with empty parts",
			input:   "-- :::",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "header with skip and empty parts",
			input:   "-- skip:::",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty user field",
			input:   "-- skip::ns:comment",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty namespace field",
			input:   "-- skip:user::comment",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "empty comment field",
			input:   "-- skip:user:ns:",
			want:    nil,
			wantErr: true,
		},
		{
			name:  "case-insensitive skip",
			input: "-- SKIP:user:ns:comment",
			want: &MigrationProto{
				ShouldSkip: true,
				User:       "user",
				Namespace:  "ns",
				Comment:    "comment",
			},
			wantErr: false,
		},
		{
			name:  "header with special characters in comment",
			input: "-- skip:user:ns:comment with: special! @#$%^&* chars",
			want: &MigrationProto{
				ShouldSkip: true,
				User:       "user",
				Namespace:  "ns",
				Comment:    "comment with: special! @#$%^&* chars",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMigrationHeader(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMigrationHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.ShouldSkip != tt.want.ShouldSkip {
				t.Errorf("ShouldSkip = %v, want %v", got.ShouldSkip, tt.want.ShouldSkip)
			}
			if got.User != tt.want.User {
				t.Errorf("User = %v, want %v", got.User, tt.want.User)
			}
			if got.Namespace != tt.want.Namespace {
				t.Errorf("Namespace = %v, want %v", got.Namespace, tt.want.Namespace)
			}
			if got.Comment != tt.want.Comment {
				t.Errorf("Comment = %v, want %v", got.Comment, tt.want.Comment)
			}
		})
	}
}
