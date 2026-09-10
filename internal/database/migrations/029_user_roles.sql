-- Named roles that users can combine. Existing access_role values are mapped:
-- admin/organizer -> admin, operational/employee -> corre.
CREATE TABLE IF NOT EXISTS roles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug TEXT NOT NULL UNIQUE COLLATE NOCASE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0
);

INSERT INTO roles(slug, name, description, sort_order) VALUES
    ('admin', 'Administrador', 'Cria e edita eventos, cardápios, modelos, regras e todo o restante do sistema.', 1),
    ('corre', 'Corre', 'Faz a checklist, visualiza eventos e visualiza e edita o estoque.', 2),
    ('agent', 'Agent', 'Monta layouts do zero ou a partir de eventos já cadastrados.', 3);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);
CREATE INDEX IF NOT EXISTS idx_user_roles_role ON user_roles(role_id);

INSERT OR IGNORE INTO user_roles(user_id, role_id)
SELECT u.id, r.id
FROM users u
JOIN roles r ON r.slug = CASE
    WHEN COALESCE(u.access_role, u.role) IN ('admin', 'organizer') THEN 'admin'
    ELSE 'corre'
END;
