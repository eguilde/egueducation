-- Workflow history is append-only under the application role.  Repairs must
-- be represented by compensating events, never rewritten history.
select set_config('app.is_super_admin', 'true', true);

create or replace function public.reject_registratura_workflow_event_mutation()
returns trigger language plpgsql as $$
begin
  raise exception 'registratura workflow events are immutable' using errcode = '55000';
end $$;

drop trigger if exists trg_registratura_workflow_events_immutable on registratura_document_workflow_events;
create trigger trg_registratura_workflow_events_immutable
before update or delete on registratura_document_workflow_events
for each row execute function public.reject_registratura_workflow_event_mutation();
