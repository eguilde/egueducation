-- A committee member must belong to the same institution as its parent
-- committee. The original single-column foreign key proves only that the
-- committee UUID exists and permits privileged writes to cross tenant scope.

create unique index if not exists education_committees_institution_id_id_uq
	on public.education_committees (institution_id, id);

alter table public.education_committee_members
	drop constraint if exists education_committee_members_committee_id_fkey;

alter table public.education_committee_members
	add constraint education_committee_members_parent_institution_fk
	foreign key (institution_id, committee_id)
	references public.education_committees (institution_id, id)
	on delete cascade;

-- The constraint is deliberately validated immediately. Any legacy
-- cross-institution reference aborts this transactional migration rather than
-- preserving unsafe data behind a NOT VALID constraint.
