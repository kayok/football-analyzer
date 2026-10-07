CREATE TABLE sync_status (
 id integer PRIMARY KEY CHECK (id = 1),
 data jsonb NOT NULL
);
INSERT INTO sync_status(id,data) VALUES(1,'{"state":"idle","message":"ยังไม่ได้ซิงก์"}');
