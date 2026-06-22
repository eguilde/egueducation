create table if not exists registre (
	id bigserial primary key,
	nume text not null,
	prefix_nr text not null default 'REG',
	nr_inceput integer not null default 1,
	nr_curent text not null default '',
	nr_urmator text not null default '',
	data_resetare date,
	tip_registru text not null default 'general',
	is_default boolean not null default false,
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	check (tip_registru in ('general', 'intrare', 'iesire', 'special'))
);

alter table registratura_documents
	add column if not exists registru_id bigint references registre(id) on delete restrict;

insert into registre (nume, prefix_nr, nr_inceput, nr_curent, nr_urmator, tip_registru, is_default)
select 'Registru General', 'REG', 1, '0012', '0013', 'general', true
where not exists (select 1 from registre where nume = 'Registru General');

insert into registre (nume, prefix_nr, nr_inceput, nr_curent, nr_urmator, tip_registru, is_default)
select 'Registru Intrări', 'IN', 1, '0000', '0001', 'intrare', false
where not exists (select 1 from registre where nume = 'Registru Intrări');

insert into registre (nume, prefix_nr, nr_inceput, nr_curent, nr_urmator, tip_registru, is_default)
select 'Registru Ieșiri', 'OUT', 1, '0000', '0001', 'iesire', false
where not exists (select 1 from registre where nume = 'Registru Ieșiri');

update registratura_documents
set registru_id = (
	select id
	from registre
	where is_default = true
	limit 1
)
where registru_id is null;

update registre r
set nr_curent = coalesce((
	select to_char(count(*)::int, 'FM0000')
	from registratura_documents d
	where d.registru_id = r.id
), '0000'),
nr_urmator = coalesce((
	select to_char(count(*)::int + 1, 'FM0000')
	from registratura_documents d
	where d.registru_id = r.id
), '0001'),
updated_at = now();
