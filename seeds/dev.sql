BEGIN;

-- Remove os dados criados pelo seed anterior.
-- A ordem respeita as foreign keys.
DELETE FROM match_results;
DELETE FROM matches;
DELETE FROM match_invites;
DELETE FROM user_locations;
DELETE FROM users;

-- Users
INSERT INTO users (id, name)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'Alice'),
    ('22222222-2222-2222-2222-222222222222', 'Bob'),
    ('33333333-3333-3333-3333-333333333333', 'Carlos'),
    ('44444444-4444-4444-4444-444444444444', 'Daniel');

-- User locations
INSERT INTO user_locations (user_id, location)
VALUES
    (
        '11111111-1111-1111-1111-111111111111',
        ST_SetSRID(ST_MakePoint(-37.0731, -10.9472), 4326)::geography
    ),
    (
        '22222222-2222-2222-2222-222222222222',
        ST_SetSRID(ST_MakePoint(-37.0735, -10.9475), 4326)::geography
    ),
    (
        '33333333-3333-3333-3333-333333333333',
        ST_SetSRID(ST_MakePoint(-37.0740, -10.9480), 4326)::geography
    ),
    (
        '44444444-4444-4444-4444-444444444444',
        ST_SetSRID(ST_MakePoint(-37.1000, -11.0000), 4326)::geography
    );

-- Pending invites
INSERT INTO match_invites (id, sender_id, receiver_id, status)
VALUES
    (
        'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa',
        '11111111-1111-1111-1111-111111111111',
        '44444444-4444-4444-4444-444444444444',
        'pending'
    ),
    (
        'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb',
        '33333333-3333-3333-3333-333333333333',
        '44444444-4444-4444-4444-444444444444',
        'pending'
    );

-- Rejected invite
INSERT INTO match_invites (id, sender_id, receiver_id, status)
VALUES
    (
        'cccccccc-cccc-cccc-cccc-cccccccccccc',
        '22222222-2222-2222-2222-222222222222',
        '11111111-1111-1111-1111-111111111111',
        'rejected'
    );

-- Finished match: Alice wins, Bob loses
INSERT INTO matches (
    id,
    player_1_id,
    player_2_id,
    player_1_finished,
    player_2_finished,
    status,
    finished_at
)
VALUES (
    'aaaaaaaa-1111-1111-1111-aaaaaaaaaaaa',
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222222',
    true,
    true,
    'finished',
    NOW()
);

INSERT INTO match_results (match_id, player_id, result)
VALUES
    (
        'aaaaaaaa-1111-1111-1111-aaaaaaaaaaaa',
        '11111111-1111-1111-1111-111111111111',
        'win'
    ),
    (
        'aaaaaaaa-1111-1111-1111-aaaaaaaaaaaa',
        '22222222-2222-2222-2222-222222222222',
        'loss'
    );

-- Finished match: Alice and Carlos draw
INSERT INTO matches (
    id,
    player_1_id,
    player_2_id,
    player_1_finished,
    player_2_finished,
    status,
    finished_at
)
VALUES (
    'bbbbbbbb-2222-2222-2222-bbbbbbbbbbbb',
    '11111111-1111-1111-1111-111111111111',
    '33333333-3333-3333-3333-333333333333',
    true,
    true,
    'finished',
    NOW()
);

INSERT INTO match_results (match_id, player_id, result)
VALUES
    (
        'bbbbbbbb-2222-2222-2222-bbbbbbbbbbbb',
        '11111111-1111-1111-1111-111111111111',
        'draw'
    ),
    (
        'bbbbbbbb-2222-2222-2222-bbbbbbbbbbbb',
        '33333333-3333-3333-3333-333333333333',
        'draw'
    );

-- Finished match: Bob and Carlos both declare win
INSERT INTO matches (
    id,
    player_1_id,
    player_2_id,
    player_1_finished,
    player_2_finished,
    status,
    finished_at
)
VALUES (
    'cccccccc-3333-3333-3333-cccccccccccc',
    '22222222-2222-2222-2222-222222222222',
    '33333333-3333-3333-3333-333333333333',
    true,
    true,
    'finished',
    NOW()
);

INSERT INTO match_results (match_id, player_id, result)
VALUES
    (
        'cccccccc-3333-3333-3333-cccccccccccc',
        '22222222-2222-2222-2222-222222222222',
        'win'
    ),
    (
        'cccccccc-3333-3333-3333-cccccccccccc',
        '33333333-3333-3333-3333-333333333333',
        'win'
    );

-- Active match: Daniel and Alice
INSERT INTO matches (
    id,
    player_1_id,
    player_2_id,
    status
)
VALUES (
    'dddddddd-4444-4444-4444-dddddddddddd',
    '44444444-4444-4444-4444-444444444444',
    '11111111-1111-1111-1111-111111111111',
    'active'
);

COMMIT;