-- User table
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add an index on the name columns for faster searches
CREATE INDEX idx_users_names ON users(first_name, last_name);

-- Audit log table
CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    trigerred_by BIGINT REFERENCES users(id),
    trigerred_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    details TEXT NOT NULL
);

-- Add an index on trigerred_at for faster time-based queries
CREATE INDEX idx_audit_log_time ON audit_log(trigerred_at);

-- Function to automatically update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update the updated_at timestamp when a user record is updated
CREATE TRIGGER update_users_modtime
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION update_modified_column();

-- Function to log user changes to audit_log
CREATE OR REPLACE FUNCTION log_user_changes()
RETURNS TRIGGER AS $$
DECLARE
    change_details TEXT;
    user_id BIGINT;
BEGIN
    -- Get the user ID from the application context
    -- The application should set this using: SET LOCAL app.users_id = 'user_id_value';
    user_id := current_setting('app.users_id', TRUE);
    
    -- If user_id is not set, use NULL (system operation)
    IF user_id IS NULL THEN
        user_id := NULL;
    END IF;
    
    -- Create details text based on operation type
    IF TG_OP = 'INSERT' THEN
        change_details := 'Created new user: ' || NEW.first_name || ' ' || NEW.last_name || ' (ID: ' || NEW.id || ')';
    ELSIF TG_OP = 'UPDATE' THEN
        change_details := 'Updated user: ' || NEW.first_name || ' ' || NEW.last_name || ' (ID: ' || NEW.id || ')';
        
        -- Add details about what changed
        IF OLD.first_name <> NEW.first_name THEN
            change_details := change_details || ', first_name: ' || OLD.first_name || ' -> ' || NEW.first_name;
        END IF;
        
        IF OLD.last_name <> NEW.last_name THEN
            change_details := change_details || ', last_name: ' || OLD.last_name || ' -> ' || NEW.last_name;
        END IF;
    END IF;
    
    -- Insert into audit_log
    INSERT INTO audit_log (trigerred_by, details)
    VALUES (user_id, change_details);
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to log user changes after INSERT or UPDATE
CREATE TRIGGER log_users_changes
AFTER INSERT OR UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION log_user_changes();
