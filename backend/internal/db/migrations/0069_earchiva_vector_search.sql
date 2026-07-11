create or replace function archive_vector_dot(a double precision[], b double precision[])
returns double precision
language sql
immutable
strict
as $$
	select case
		when coalesce(array_length(a, 1), 0) = 0 or coalesce(array_length(b, 1), 0) = 0 then 0
		when array_length(a, 1) <> array_length(b, 1) then 0
		else (
			select coalesce(sum(a[i] * b[i]), 0)
			from generate_subscripts(a, 1) as g(i)
		)
	end;
$$;

alter table archive_document_versions
	add column if not exists search_embedding double precision[] not null default '{}'::double precision[];
