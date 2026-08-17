alter table education_meetings
    add column if not exists chairperson_user_id uuid,
    add column if not exists secretary_user_id uuid;

alter table education_governance_memberships
    add column if not exists app_user_id uuid;

do $$ begin
    alter table education_meetings
        add constraint education_meetings_chairperson_user_fk
        foreign key (chairperson_user_id) references app_users(id) on delete restrict;
exception when duplicate_object then null;
end $$;

do $$ begin
    alter table education_meetings
        add constraint education_meetings_secretary_user_fk
        foreign key (secretary_user_id) references app_users(id) on delete restrict;
exception when duplicate_object then null;
end $$;

do $$ begin
    alter table education_governance_memberships
        add constraint education_governance_memberships_app_user_fk
        foreign key (app_user_id) references app_users(id) on delete restrict;
exception when duplicate_object then null;
end $$;

create index if not exists education_meetings_actor_ids_idx
    on education_meetings(institution_id, chairperson_user_id, secretary_user_id);

create index if not exists education_governance_memberships_app_user_idx
    on education_governance_memberships(institution_id, app_user_id, school_year, organism);

-- Existing rows are intentionally not backfilled from display names. They remain
-- ineligible for privileged contextual actions until an administrator assigns the
-- immutable identity explicitly.
