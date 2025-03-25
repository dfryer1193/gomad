-- :dfryer:migrations:Create migrations table(s)
CREATE TABLE migrations (
    id SERIAL PRIMARY KEY,
    namespace VARCHAR(50) NOT NULL,
    user VARCHAR(50),
    comment TEXT,
    ddl TEXT,
    completed TIMESTAMP,
);

-- :dfryer:migrations:Create webhook secrets table
CREATE TABLE webhook_secrets (
    repo_name VARCHAR(255) PRIMARY KEY,
    secret VARCHAR(64) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
