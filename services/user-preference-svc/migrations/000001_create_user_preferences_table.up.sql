CREATE TABLE IF NOT EXISTS user_preferences (
    user_preference_id UUID NOT NULL DEFAULT gen_random_uuid(),
    client_id          VARCHAR(100) NOT NULL,
    external_user_id   VARCHAR(255) NOT NULL,
    channels           JSONB NOT NULL DEFAULT '{}'::jsonb,
    types              JSONB NOT NULL DEFAULT '{}'::jsonb,
    do_not_disturb     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_preference_id)
);

CREATE UNIQUE INDEX idx_client_external
ON user_preferences (client_id, external_user_id);
