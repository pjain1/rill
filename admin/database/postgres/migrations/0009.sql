--INSERT INTO project_roles (name, read_project, manage_project, read_prod, read_prod_status, manage_prod, read_dev, read_dev_status, manage_dev, read_project_members, manage_project_members)
--VALUES ('restricted', true, false, true, false, false, false, false, false, false, false);

CREATE TABLE restricted_project_roles (
    id UUID DEFAULT uuid_generate_v4() PRIMARY KEY,
    name TEXT NOT NULL,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    attributes JSONB NOT NULL -- TODO separate table with type information
);

CREATE UNIQUE INDEX restricted_project_roles_name_idx ON restricted_project_roles (project_id, lower(name));

-- Add a restricted role to existing projects
INSERT INTO restricted_project_roles (name, project_id, attributes) SELECT 'restricted', id, '{}' FROM projects;

CREATE TABLE users_restricted_project_roles (
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    project_role_id UUID NOT NULL REFERENCES restricted_project_roles (id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, project_id)
);

CREATE INDEX users_restricted_project_roles_project_user_idx ON users_restricted_project_roles (project_id, user_id);

CREATE TABLE usergroups_restricted_projects_roles (
    usergroup_id UUID NOT NULL REFERENCES usergroups (id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    project_role_id UUID NOT NULL REFERENCES restricted_project_roles (id) ON DELETE CASCADE,
    PRIMARY KEY (usergroup_id, project_id)
);

CREATE INDEX usergroups_restricted_projects_roles_project_usergroup_idx ON usergroups_restricted_projects_roles (project_id, usergroup_id);

ALTER TABLE project_invites ADD COLUMN project_role_id UUID REFERENCES restricted_project_roles (id) ON DELETE CASCADE;