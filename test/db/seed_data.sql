INSERT INTO users (uuid, id, email, role, created_at, updated_at)
VALUES 
    ('018f0000-0000-7000-8000-000000000001', 'test_pilot', 'pilot@example.com', 'user', NOW() - INTERVAL '2 days', NOW()),
    ('018f0000-0000-7000-8000-000000000010', 'alex', 'alex@quadsmith.dev', 'user', NOW() - INTERVAL '2 days', NOW())
ON CONFLICT (uuid) DO NOTHING;

INSERT INTO user_profiles (uuid, user_uuid, avatar_url, bio, updated_at)
VALUES 
    ('018f0000-0000-7000-8000-000000000002', '018f0000-0000-7000-8000-000000000001', '', 'E2E Test Account', NOW()),
    ('018f0000-0000-7000-8000-000000000011', '018f0000-0000-7000-8000-000000000010', '', 'Community Builder', NOW())
ON CONFLICT (user_uuid) DO NOTHING;

INSERT INTO user_emails (uuid, user_uuid, email, is_primary, created_at)
VALUES 
    ('018f0000-0000-7000-8000-000000000003', '018f0000-0000-7000-8000-000000000001', 'pilot@example.com', true, NOW() - INTERVAL '2 days'),
    ('018f0000-0000-7000-8000-000000000012', '018f0000-0000-7000-8000-000000000010', 'alex@quadsmith.dev', true, NOW() - INTERVAL '2 days')
ON CONFLICT (email) DO NOTHING;

INSERT INTO user_reputations (uuid, user_uuid, score, updated_at)
VALUES
    ('018f0000-0000-7000-8000-000000000004', '018f0000-0000-7000-8000-000000000001', 150, NOW()),
    ('018f0000-0000-7000-8000-000000000013', '018f0000-0000-7000-8000-000000000010', 250, NOW())
ON CONFLICT (user_uuid) DO NOTHING;
