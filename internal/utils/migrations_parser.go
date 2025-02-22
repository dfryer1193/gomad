package utils

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"strings"
)

type MigrationProto struct {
	ShouldSkip bool
	User       string
	Namespace  string
	Comment    string
	DDL        string
	Signature  uint64
}

// ParseSQL parses a SQL file content into a slice of MigrationProto structs.
// The SQL file should have migrations in the format:
// -- skip?:user:namespace:comment
// SQL statements...
func ParseSQL(content string) ([]MigrationProto, error) {
	var migrations []MigrationProto
	var currentMigration *MigrationProto
	var ddlBuilder strings.Builder
	var foundFirstHeader bool

	scanner := bufio.NewScanner(strings.NewReader(content))

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		if trimmedLine == "" {
			continue
		}

		if strings.HasPrefix(line, "--") {
			if currentMigration != nil && ddlBuilder.Len() == 0 {
				return nil, fmt.Errorf("invalid migration: migration header without SQL content: %s", currentMigration.Comment)
			}

			if currentMigration != nil && ddlBuilder.Len() > 0 {
				currentMigration.DDL = strings.TrimSpace(ddlBuilder.String())
				migrations = append(migrations, *currentMigration)
				ddlBuilder.Reset()
			}

			// Parse the header line
			migration, err := parseMigrationHeader(line)
			if err != nil {
				return nil, err
			}
			migration.Signature = generateSignature(line)
			currentMigration = migration
			foundFirstHeader = true
			continue
		}

		if !foundFirstHeader {
			return nil, fmt.Errorf("invalid migration: missing migration header")
		}

		if currentMigration != nil {
			ddlBuilder.WriteString(line)
			ddlBuilder.WriteString("\n")
		}
	}

	// Add the last migration if exists
	if currentMigration != nil && ddlBuilder.Len() > 0 {
		currentMigration.DDL = strings.TrimSpace(ddlBuilder.String())
		migrations = append(migrations, *currentMigration)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading SQL content: %w", err)
	}

	return migrations, nil
}

// parseMigrationHeader parses a comment line in the format "-- skip?:user:namespace:comment"
func parseMigrationHeader(line string) (*MigrationProto, error) {
	input := strings.Clone(line)

	input = strings.TrimPrefix(input, "--")
	input = strings.TrimSpace(input)

	userStart := strings.Index(input, ":")
	if userStart < 0 {
		return nil, fmt.Errorf("invalid migration header: not enough parts: %s", line)
	}

	namespaceStart := strings.Index(input[userStart+1:], ":")
	if namespaceStart < 0 {
		return nil, fmt.Errorf("invalid migration header: not enough parts: %s", line)
	}
	namespaceStart += userStart + 1

	commentStart := strings.Index(input[namespaceStart+1:], ":")
	if commentStart < 0 {
		return nil, fmt.Errorf("invalid migration header: not enough parts: %s", line)
	}
	commentStart += namespaceStart + 1

	shouldSkip := strings.ToLower(strings.TrimSpace(input[:userStart])) == "skip"
	user := strings.TrimSpace(input[userStart+1 : namespaceStart])
	namespace := strings.TrimSpace(input[namespaceStart+1 : commentStart])
	comment := strings.TrimSpace(input[commentStart+1:])

	if user == "" {
		return nil, fmt.Errorf("invalid migration header: user is empty: %s", line)
	}

	if namespace == "" {
		return nil, fmt.Errorf("invalid migration header: namespace is empty: %s", line)
	}

	if comment == "" {
		return nil, fmt.Errorf("invalid migration header: comment is empty: %s", line)
	}

	return &MigrationProto{
		ShouldSkip: shouldSkip,
		User:       user,
		Namespace:  namespace,
		Comment:    comment,
	}, nil
}

func generateSignature(header string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(header))
	return h.Sum64()
}
