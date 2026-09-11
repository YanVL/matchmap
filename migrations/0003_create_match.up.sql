CREATE TABLE matches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_1_id UUID NOT NULL REFERENCES users(id),
    player_2_id UUID NOT NULL REFERENCES users(id),

    player_1_finished BOOLEAN NOT NULL DEFAULT false,
    player_2_finished BOOLEAN NOT NULL DEFAULT false,

    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,

    CHECK (player_1_id <> player_2_id),
    CHECK (status IN ('active', 'finished'))
);