-- +migrate Up
-- Sample Users
INSERT INTO users (id, username, email, password, first_name, last_name, date_of_birth, privacy, bio) VALUES
('u1', 'john_doe', 'john@example.com', '$2a$10$XgXB8ZUJxuYGv5/rnD9Ym.fCgg7iTJJIyEDJVY9.hKGQzqkTtQT2', 'John', 'Doe', '1990-01-01', 'public', 'Software developer'),
('u2', 'jane_smith', 'jane@example.com', '$2a$10$XgXB8ZUJxuYGv5/rnD9Ym.fCgg7iTJJIyEDJVY9.hKGQzqkTtQT2', 'Jane', 'Smith', '1992-05-15', 'public', 'UX Designer'),
('u3', 'bob_wilson', 'bob@example.com', '$2a$10$XgXB8ZUJxuYGv5/rnD9Ym.fCgg7iTJJIyEDJVY9.hKGQzqkTtQT2', 'Bob', 'Wilson', '1988-12-20', 'private', 'Product Manager'),
('u4', 'alice_jones', 'alice@example.com', '$2a$10$XgXB8ZUJxuYGv5/rnD9Ym.fCgg7iTJJIyEDJVY9.hKGQzqkTtQT2', 'Alice', 'Jones', '1995-08-30', 'public', 'Graphic Designer'),
('u5', 'charlie_brown', 'charlie@example.com', '$2a$10$XgXB8ZUJxuYGv5/rnD9Ym.fCgg7iTJJIyEDJVY9.hKGQzqkTtQT2', 'Charlie', 'Brown', '1991-03-25', 'public', 'Marketing Specialist');

INSERT INTO posts (id, user_id, author, title, content, creation_date, status)
VALUES
('p1', 'u1', 'john_doe', 'Welcome to Tech Enthusiasts', 'Lets discuss the latest in tech!', DATETIME('now', '-5 days'), 'public'),
('p2', 'u2', 'jane_smith', 'Design Trends 2024', 'What are your favorite new design trends?',DATETIME('now', '-4 days'), 'public'),
('p3', 'u3', 'bob_wilson', 'Productivity Tips', 'Share your best productivity hacks.' ,DATETIME('now', '-3 days'), 'private'),
('p4', 'u4', 'alice_jones', 'Creative Inspiration', 'Where do you find inspiration?',DATETIME('now', '-2 days'), 'public'),
('p5', 'u5', 'charlie_brown', 'Marketing Strategies', 'Lets talk about digital marketing.',DATETIME('now', '-1 day'), 'public');

INSERT INTO comments (id, post_id, author, content, creation_date)
VALUES
('c1', 'p1', 'jane_smith', 'Great topic, John!', DATETIME('now', '-4 days')),
('c2', 'p1', 'bob_wilson', 'Looking forward to the discussion.', DATETIME('now', '-3 days')),
('c3', 'p2', 'alice_jones', 'Love these trends!', DATETIME('now', '-2 days')),
('c4', 'p3', 'charlie_brown', 'My best tip: take breaks!', DATETIME('now', '-1 day')),
('c5', 'p4', 'john_doe', 'Nature is my biggest inspiration.', DATETIME('now', '-12 hours'));


-- Sample Groups
INSERT INTO groups (id, creator_id, title, description) VALUES
('g1', 'u1', 'Tech Enthusiasts', 'A group for discussing the latest in technology'),
('g2', 'u2', 'Design Hub', 'Share and discuss design trends and inspiration'),
('g3', 'u3', 'Product Management', 'Tips and discussions about product management'),
('g4', 'u4', 'Creative Corner', 'A space for creative minds to connect'),
('g5', 'u5', 'Digital Marketing', 'Strategies and trends in digital marketing');

-- Sample Group Members
INSERT INTO group_members (group_id, user_id, status, is_admin) VALUES
('g1', 'u1', 'accepted', 1),
('g1', 'u2', 'accepted', 0),
('g1', 'u3', 'pending', 0),
('g2', 'u2', 'accepted', 1),
('g2', 'u4', 'accepted', 0),
('g3', 'u3', 'accepted', 1),
('g3', 'u1', 'accepted', 0),
('g4', 'u4', 'accepted', 1),
('g4', 'u5', 'pending', 0),
('g5', 'u5', 'accepted', 1);

-- Sample Group Posts
INSERT INTO group_posts (id, group_id, user_id, title, content, creation_date) VALUES
('gp1', 'g1', 'u1', 'Latest Tech Trends', 'Discussing the emerging technologies of 2024', DATETIME('now', '-2 days')),
('gp2', 'g1', 'u2', 'AI Developments', 'Recent breakthroughs in artificial intelligence', DATETIME('now', '-1 day')),
('gp3', 'g2', 'u2', 'UI Design Principles', 'Essential principles for modern UI design', DATETIME('now', '-3 days')),
('gp4', 'g3', 'u3', 'Agile Management', 'Best practices in agile product management', DATETIME('now', '-4 days')),
('gp5', 'g4', 'u4', 'Creative Inspiration', 'Finding inspiration in everyday things', DATETIME('now', '-1 hour'));

-- Sample Group Comments
INSERT INTO group_comments (id, group_post_id, author, content, creation_date) VALUES
('gc1', 'gp1', 'jane_smith', 'Great insights on tech trends!', DATETIME('now', '-1 day')),
('gc2', 'gp1', 'bob_wilson', 'Would love to learn more about AI applications', DATETIME('now', '-12 hours')),
('gc3', 'gp2', 'john_doe', 'Amazing developments in AI lately', DATETIME('now', '-6 hours')),
('gc4', 'gp3', 'alice_jones', 'These principles are very helpful', DATETIME('now', '-2 days')),
('gc5', 'gp4', 'charlie_brown', 'Great tips for agile management!', DATETIME('now', '-3 days'));

-- Sample Events
INSERT INTO events (id, title, description, event_datetime, location, creator_id, group_id, creation_date) VALUES
('e1', 'Tech Meetup 2024', 'Annual technology meetup', DATETIME('now', '+30 days'), 'Tech Hub Center', 'u1', 'g1', DATETIME('now', '-10 days')),
('e2', 'Design Workshop', 'Hands-on UI/UX workshop', DATETIME('now', '+15 days'), 'Design Studio', 'u2', 'g2', DATETIME('now', '-5 days')),
('e3', 'Product Strategy Session', 'Discussion on product strategy', DATETIME('now', '+45 days'), 'Innovation Center', 'u3', 'g3', DATETIME('now', '-7 days')),
('e4', 'Creative Networking', 'Network with creative professionals', DATETIME('now', '+20 days'), 'Art Gallery', 'u4', 'g4', DATETIME('now', '-3 days')),
('e5', 'Marketing Conference', 'Digital marketing trends conference', DATETIME('now', '+60 days'), 'Conference Center', 'u5', 'g5', DATETIME('now', '-15 days'));

-- Sample Event Responses
INSERT INTO event_responses (id, user_id, event_id, option, response_date) VALUES
('er1', 'u2', 'e1', 1, DATETIME('now', '-9 days')),
('er2', 'u3', 'e1', 1, DATETIME('now', '-8 days')),
('er3', 'u4', 'e2', 1, DATETIME('now', '-4 days')),
('er4', 'u1', 'e3', -1, DATETIME('now', '-6 days')),
('er5', 'u5', 'e4', 1, DATETIME('now', '-2 days'));

-- Sample Followers
INSERT INTO Followers (id, follower_id, followed_id, status) VALUES
('f1', 'u1', 'u2', 'accepted'),
('f2', 'u2', 'u1', 'accepted'),
('f3', 'u3', 'u1', 'pending'),
('f4', 'u4', 'u2', 'accepted'),
('f5', 'u5', 'u3', 'accepted');

-- Sample Notifications
INSERT INTO notifications (id, user_id, sender_id, type, content, is_read, created_at, related_entity_id, related_entity_type) VALUES
('n1', 'u2', 'u1', 'group_invite', 'John Doe invited you to join Tech Enthusiasts', 0, DATETIME('now', '-1 day'), 'g1', 'group'),
('n2', 'u3', 'u2', 'event_invite', 'Jane Smith invited you to Design Workshop', 0, DATETIME('now', '-2 days'), 'e2', 'event'),
('n3', 'u1', 'u3', 'follow_request', 'Bob Wilson wants to follow you', 1, DATETIME('now', '-3 days'), 'f3', 'follow'),
('n4', 'u4', 'u5', 'group_post', 'New post in Creative Corner', 0, DATETIME('now', '-4 hours'), 'gp5', 'post'),
('n5', 'u5', 'u4', 'comment', 'Alice Jones commented on your post', 1, DATETIME('now', '-1 hour'), 'gc4', 'comment'); 