-- Costesti parity uses DOCUMENT for an ordinary registry entry and MULTIPLU
-- for a generated batch. Keep the historic specialised types available.
insert into app_nomenclatures(domain, code, label_ro, label_en, active, sort_order)
values
  ('registratura_document_type', 'DOCUMENT', 'Document', 'Document', true, 1),
  ('registratura_document_type', 'MULTIPLU', 'Multiplu', 'Batch', true, 2)
on conflict (domain, code) do update
set label_ro = excluded.label_ro,
    label_en = excluded.label_en,
    active = true,
    sort_order = excluded.sort_order;
