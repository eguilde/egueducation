# Specificație executabilă — cutover policy v2 / migrarea 0146

## Scop și limită de siguranță

Migrarea `0146` realizează backfill-ul determinist dintre modelul legacy creat
de `0135` și bridge-ul v2 creat de `0145`. Migrarea este relansabilă,
serializată pe `tenant_code + institution_id` și nu activează automat runtime-ul
v2. Imediat după `0146`, toate scope-urile trebuie să rămână în faza `legacy`.

Trecerea `legacy → dual` este permisă numai după ce:

1. writer-ul runtime creează atomic perechea v1/v2, identity map și bindings;
2. o reconciliere nouă acoperă toate scrierile realizate după migrare;
3. nu există findings blocante;
4. numărătorile și manifestele calculate la tranziție sunt identice cu ultima
   reconciliere;
5. rollback rehearsal și shadow comparison sunt demonstrate.

Tranzițiile `dual → v2 → contracted` aparțin migrării `0147` și sunt refuzate
explicit de contractul `0146`.

## Invariante

- Backfill-ul nu ghicește forma juridică, surse, checksum-uri, persoane juridice,
  cult, fondator, finanțator sau protocoale.
- Identitatea v1/v2 nu este derivată din numărul versiunii.
- Orice asociere ambiguă produce finding `blocking` și nu produce mapare.
- Snapshoturile și evaluările istorice existente nu sunt actualizate.
- Orice import istoric nou este marcat `decision_kind=legacy_import` și nu poate
  autoriza operații runtime.
- Consumatorii păstrează FK-ul legacy și primesc FK-ul v2 numai dacă maparea este
  exactă și neconflictuală.
- O eroare într-un scope nu mută scope-ul în `dual`; findings sunt date auditate,
  nu motive pentru a inventa o reconciliere reușită.

## 1. Reconciliere și controlul fazei

Se adaugă `school_policy_cutover_reconciliations`, cu `FORCE RLS`, evidență
imuabilă și următoarele câmpuri obligatorii:

- scope tenant/instituție și `run_id`;
- source watermark;
- numărători explicite pentru profile, assignments, overrides, evaluations și
  consumatori;
- checksum SHA-256 pentru manifestul sursă și manifestul mapărilor;
- numărul findings-urilor blocante;
- stare `matched` sau `blocked`.

`school_policy_cutover_state.last_reconciliation_id` referă compozit
reconcilierea exactă. Triggerul de tranziție aplică optimistic concurrency prin
`expected_version`, permite în `0146` numai `legacy → dual` și recalculează live
numărătorile înainte de acceptare.

Backfill-ul rulează printr-o funcție `SECURITY INVOKER`, accesibilă numai rolului
de migrare. Funcția folosește advisory lock per scope și lock-uri de citire pe
tabelele legacy pe durata watermarkului și reconcilierii.

## 2. Profile v1 → v2

Ordinea de lucru este deterministă: `legacy.version, legacy.id`.

### 2.1 Perechi dual-write existente

O pereche este acceptată numai dacă este unică în ambele direcții și coincide
exact pe:

- scope;
- aceeași tranzacție (`created_at`, `created_by_subject`);
- status;
- forma juridică normalizată (`confessional → private`);
- intervalul efectiv;
- personalitate juridică, identificator fiscal, bază contabilă, TVA și
  obligativitatea Trezoreriei.

Zero candidați pornește importul legacy. Mai mult de un candidat produce
`profile_identity_ambiguous`.

### 2.2 Import legacy

- ID-ul fizic v2 este UUID determinist din namespace + ID legacy.
- Versiunile fizice v2 sunt alocate peste maximul curent, în ordinea stabilă a
  rândurilor legacy.
- Identitatea API rămâne ID-ul și versiunea legacy.
- `unclassified` devine v2 `unclassified`, cu `legal_form` și `effective_from`
  nule.
- `confessional` devine `private`, dar produce
  `confessional_overlay_reconstruction_required`; nu se creează automat cult,
  protocol sau parte juridică.
- Un profil approved/active care se suprapune peste un profil v2 nemapat produce
  `profile_effective_overlap` și nu este importat.

Orice profil v2 rămas fără identity map produce `unmapped_v2_profile`.

## 3. Proiecție și surse

`school_institution_profile_api_projection` copiază exact câmpurile legacy și
este imuabilă. O proiecție existentă diferită produce
`profile_projection_conflict`; nu se execută `UPDATE`.

Pentru un `source_reference` legacy nenul se creează numai o sursă `draft`, de
tip `other`, cu citarea exactă. URL-ul, checksum-ul și verificarea rămân goale.
Un profil approved/active importat primește `legacy_source_unverified`; dacă
referința lipsește, `legacy_source_missing`.

Numele legacy de fondator, finanțator sau ordonator rămân numai în proiecția de
compatibilitate. Ele nu creează `app_parties`. Când sunt necesare pentru un
profil utilizabil, se emite `legacy_party_role_reconstruction_required`.

## 4. Assignments, bindings și overrides

Fiecare assignment rezolvabil primește un binding v2 determinist:

- același pack ID, cod și versiune, verificate prin FK-ul exact din `0145`;
- profilul, versiunea și seria exacte;
- intervalul este intersecția intervalului assignmentului cu intervalul
  profilului;
- un assignment `active` al unui profil `superseded` devine binding
  `superseded`;
- `legal-form.ro.confessional` folosește `assignment_kind=confessional`;
- fereastra vidă, pack-ul inconsistent sau profilul nemapat blochează importul.

Overrides păstrează exact valoarea, statusul, versiunea, aprobatorul și momentele
legacy. Un override approved pe o cheie care nu aparține `configurable_keys`
produce finding blocant și nu este importat.

## 5. Evaluări istorice și proveniență

Pentru fiecare evaluare v1 se creează un input și o evaluare v2 noi și imuabile.
Nu se modifică inputurile v2 create înainte de `0145`.

Inputul folosește:

- `operation_code=legacy.policy_evaluation`;
- `decision_kind=legacy_import`;
- `engine_version=v1-import`;
- `effective_on=evaluated_at::date`;
- context cu ID/versiune profil legacy, lista exactă de pack-uri și checksum-ul
  evaluării legacy;
- checksum canonic calculat din JSONB.

Pentru fiecare pack trebuie să existe exact un binding aplicabil la data
evaluării. Overrides se reconstruiesc exclusiv din ultimul snapshot
`app_entity_versions` cu `changed_at <= evaluated_at`, ordonat stabil. Nu se
folosește starea curentă drept fallback. Istoricul incomplet produce
`override_history_unreconstructable`.

Evaluarea v2 păstrează capabilities, warnings, actorul și momentul legacy;
`allowed = NOT blocked`, iar obligations este lista goală. Identity map-ul
folosește `conversion_kind=historical_import`.

Inputurile v2 preexistente fără `effective_on` și inputurile cu mai multe
evaluări sunt findings blocante. Pentru date curate se adaugă unicitatea reală
`(tenant_code, institution_id, input_id)`; în caz contrar, un trigger previne
apariția altor duplicate până la remediere.

## 6. Consumatori

Evaluarea v2 este propagată numai prin maparea exactă către:

1. `education_publications`;
2. `school_contracts`;
3. `school_contract_obligations`;
4. `school_utility_points`;
5. `school_utility_readings`;
6. `school_utility_invoices`;
7. `school_compliance_obligations`;
8. `school_compliance_inspections`;
9. `school_compliance_corrective_actions`.

Un `policy_evaluation_v2_id` existent și diferit produce
`consumer_v2_evaluation_conflict` și nu este suprascris. Pentru actualizarea
bounded sunt suspendate numai `trg_education_publications_official_immutability`
și `school_contract_history`, apoi sunt reactivate în aceeași tranzacție. Toate
FK-urile și check-urile `NOT VALID` din `0145` se validează după backfill.

## 7. Interogări obligatorii de acceptare

Preflight-ul read-only este implementat în
`backend/internal/db/school_policy_cutover_preflight.go`. El derivă scope-ul
exclusiv din sesiunea PostgreSQL și nu acceptă tenantul sau instituția ca input
de la client. Rezultatul este structural; `ReadyForDual()` nu substituie
reconcilierea, shadow comparison sau aprobarea operațională.

Înaintea tranziției la `dual`, toate interogările următoare trebuie să întoarcă
zero rânduri sau count zero:

```sql
select count(*)
from school_institution_profiles p
left join school_profile_cutover_identity i
  on (i.tenant_code,i.institution_id,i.legacy_profile_id,i.legacy_profile_version)
   = (p.tenant_code,p.institution_id,p.id,p.version)
where i.id is null;

select count(*)
from school_policy_assignments a
left join school_operation_policy_bindings_v2 b
  on (b.tenant_code,b.institution_id,b.legacy_assignment_id)
   = (a.tenant_code,a.institution_id,a.id)
where b.id is null;

select count(*)
from school_policy_overrides o
left join school_operation_policy_overrides_v2 v
  on (v.tenant_code,v.institution_id,v.legacy_override_id)
   = (o.tenant_code,o.institution_id,o.id)
where v.id is null;

select count(*)
from school_policy_evaluations e
left join school_policy_evaluation_cutover_identity m
  on (m.tenant_code,m.institution_id,m.legacy_evaluation_id)
   = (e.tenant_code,e.institution_id,e.id)
where m.id is null;

select tenant_code,institution_id,input_id,count(*)
from school_operation_policy_evaluations_v2
group by tenant_code,institution_id,input_id
having count(*) <> 1;

select count(*)
from school_regulatory_migration_issues
where status='open' and severity='blocking';

select tenant_code,institution_id
from school_policy_cutover_state
where phase <> 'legacy';
```

Interogarea de proveniență per pack trebuie de asemenea să întoarcă zero:

```sql
select e.tenant_code,e.institution_id,e.id,p.pack_id
from school_policy_evaluations e
cross join lateral unnest(e.policy_pack_version_ids) p(pack_id)
left join school_policy_evaluation_cutover_identity m
  on (m.tenant_code,m.institution_id,m.legacy_evaluation_id)
   = (e.tenant_code,e.institution_id,e.id)
left join school_operation_policy_input_bindings ib
  on (ib.tenant_code,ib.institution_id,ib.input_id,ib.policy_pack_version_id)
   = (m.tenant_code,m.institution_id,m.input_id,p.pack_id)
where ib.id is null;
```

Pentru fiecare dintre cei nouă consumatori se verifică faptul că evaluarea v2
este cea indicată de identity map. Exemplu:

```sql
select count(*)
from school_contracts c
join school_policy_evaluation_cutover_identity m
  on (m.tenant_code,m.institution_id,m.legacy_evaluation_id)
   = (c.tenant_code,c.institution_id,c.policy_evaluation_id)
where c.policy_evaluation_v2_id is distinct from m.v2_evaluation_id;
```

## 8. Gate PostgreSQL și rollout

Migrarea nu se publică până când un test PostgreSQL disposable demonstrează:

- upgrade real `0135…0145 → 0146`, nu simulare prin eliminarea obiectelor;
- seed public, privat și confesional legacy;
- rulare repetată fără dubluri și cu reconciliation nouă;
- mapări exacte și checksum-uri stabile;
- scenarii ambigue și incomplete rămase `legacy`;
- RLS pozitiv și negativ pe noile tabele;
- trigger-ele suspendate sunt reactivate inclusiv după eroare;
- toate constrângerile `0145` sunt validate;
- runtime-ul nu poate introduce `legacy_import` și nu poate trece la `v2`.

După CI verde, rollout-ul rămâne expand–migrate–contract: `0146` produce numai
date și raport de reconciliere; `0147` schimbă sursa runtime doar după verificare
operațională și plan de rollback.
