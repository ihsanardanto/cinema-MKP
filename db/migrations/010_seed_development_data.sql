BEGIN;
-- Local test credentials only. Do not apply this migration outside development.
INSERT INTO users(email,password_hash,role)
VALUES('admin@example.com',crypt('ChangeMe123!',gen_salt('bf',12)),'admin');
INSERT INTO cities(name) VALUES('Jakarta');
INSERT INTO cinemas(city_id,name,address) VALUES(1,'MKP Cinema','Jakarta');
INSERT INTO auditoriums(cinema_id,name) VALUES(1,'Studio 1');
INSERT INTO seats(auditorium_id,label) SELECT 1,'A'||n FROM generate_series(1,10) n;
INSERT INTO movies(title,duration_minutes) VALUES('Demo Movie',120);
COMMIT;
