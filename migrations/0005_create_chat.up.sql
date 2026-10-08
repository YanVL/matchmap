CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    user_1_id UUID NOT NULL REFERENCES users(id),
    user_2_id UUID NOT NULL REFERENCES users(id),

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CHECK (user_1_id <> user_2_id)
);

CREATE UNIQUE INDEX conversations_unique_pair_idx
ON conversations (
    LEAST(user_1_id, user_2_id),
    GREATEST(user_1_id, user_2_id)
);

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    conversation_id UUID NOT NULL
        REFERENCES conversations(id)
        ON DELETE CASCADE,

    sender_id UUID NOT NULL
        REFERENCES users(id),

    content TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);