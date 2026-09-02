CREATE TABLE match_invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL REFERENCES users(id),
    receiver_id UUID NOT NULL REFERENCES users(id),
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (sender_id <> receiver_id),
    CHECK (status IN ('pending', 'accepted', 'rejected', 'finished'))

);

CREATE UNIQUE INDEX match_invites_pending_pair_idx

ON match_invites (
    LEAST(sender_id, receiver_id),
    GREATEST(sender_id, receiver_id)
)

WHERE status = 'pending';