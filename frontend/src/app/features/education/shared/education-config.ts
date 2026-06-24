import {
  EducationDashboardCardConfig,
  EducationDetailChildResourceConfig,
  EducationOption,
  EducationResourceConfig,
  EducationRouteTab,
  EducationSectionConfig,
} from './education.models';

const ORGANISM_OPTIONS: EducationOption[] = [
  { label: 'CA', value: 'ca' },
  { label: 'CP', value: 'cp' },
  { label: 'CEAC', value: 'ceac' },
  { label: 'CFDCD', value: 'cfdcd' },
];

const MEETING_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Ordinara', value: 'ordinary' },
  { label: 'Extraordinara', value: 'extraordinary' },
];

const MEETING_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'Programata', value: 'scheduled' },
  { label: 'Tinuta', value: 'held' },
  { label: 'Publicata', value: 'published' },
];

const DECISION_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'Avizata', value: 'endorsed' },
  { label: 'Aprobata', value: 'approved' },
  { label: 'Publicata', value: 'published' },
  { label: 'Blocata', value: 'blocked' },
];

const DECISION_PUBLICATION_OPTIONS: EducationOption[] = [
  { label: 'Intern', value: 'internal' },
  { label: 'In asteptarea anonimizarii', value: 'pending_anonymization' },
  { label: 'Publicat', value: 'published' },
];

const MANAGERIAL_TYPE_OPTIONS: EducationOption[] = [
  { label: 'PDI / PAS', value: 'pdi_pas' },
  { label: 'Plan managerial anual', value: 'annual_plan' },
  { label: 'RAEI', value: 'raei' },
  { label: 'Organigrama', value: 'organigram' },
  { label: 'Plan de incadrare', value: 'staffing_plan' },
  { label: 'Orar', value: 'timetable' },
  { label: 'Raport de comisie', value: 'commission_report' },
  { label: 'Portofoliu director', value: 'director_portfolio' },
  { label: 'Portofoliu director adjunct', value: 'adjunct_director_portfolio' },
];

const MANAGERIAL_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'In avizare', value: 'in_review' },
  { label: 'Aprobat', value: 'approved' },
  { label: 'Publicat', value: 'published' },
  { label: 'Arhivat', value: 'archived' },
];

const REGULATION_TYPE_OPTIONS: EducationOption[] = [
  { label: 'ROF', value: 'rof' },
  { label: 'ROI', value: 'roi' },
];

const REGULATION_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'In consultare', value: 'consultation' },
  { label: 'Avizat', value: 'endorsed' },
  { label: 'Aprobat', value: 'approved' },
  { label: 'Publicat', value: 'published' },
];

const REGULATION_APPROVAL_OPTIONS: EducationOption[] = [
  { label: 'Grup de lucru', value: 'working_group' },
  { label: 'Avizat in CP', value: 'cp_endorsed' },
  { label: 'Aprobat in CA', value: 'ca_approved' },
  { label: 'Inregistrat', value: 'registered' },
];

const MANAGERIAL_DOCUMENT_CATEGORY_OPTIONS: EducationOption[] = [
  { label: 'Diagnoza', value: 'diagnoza' },
  { label: 'Prognoza', value: 'prognoza' },
  { label: 'Evidenta', value: 'evidenta' },
  { label: 'Planificare', value: 'planificare' },
  { label: 'Raport', value: 'raport' },
  { label: 'Anexa', value: 'anexa' },
  { label: 'Hotarare', value: 'hotarare' },
  { label: 'Procedura', value: 'procedura' },
];

const MANAGERIAL_DOCUMENT_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'In avizare', value: 'in_review' },
  { label: 'Aprobat', value: 'approved' },
  { label: 'Publicat', value: 'published' },
  { label: 'Arhivat', value: 'archived' },
];

const MANAGERIAL_WORKFLOW_STAGE_OPTIONS: EducationOption[] = [
  { label: 'Elaborare', value: 'elaborare' },
  { label: 'Verificare secretariat', value: 'verificare_secretariat' },
  { label: 'Avizare in CP', value: 'avizare_cp' },
  { label: 'Aprobare in CA', value: 'aprobare_ca' },
  { label: 'Publicare', value: 'publicare' },
  { label: 'Arhivare', value: 'arhivare' },
];

const MANAGERIAL_WORKFLOW_STATUS_OPTIONS: EducationOption[] = [
  { label: 'In asteptare', value: 'pending' },
  { label: 'In lucru', value: 'in_progress' },
  { label: 'Finalizat', value: 'completed' },
  { label: 'Returnat', value: 'returned' },
  { label: 'Exceptat', value: 'waived' },
];

const REGULATION_VERSION_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'In consultare', value: 'consultation' },
  { label: 'Avizat', value: 'endorsed' },
  { label: 'Aprobat', value: 'approved' },
  { label: 'Publicat', value: 'published' },
  { label: 'Retras', value: 'retired' },
];

const REGULATION_WORKFLOW_PHASE_OPTIONS: EducationOption[] = [
  { label: 'Redactare', value: 'redactare' },
  { label: 'Consultare publica', value: 'consultare_publica' },
  { label: 'Avizare in CP', value: 'avizare_cp' },
  { label: 'Aprobare in CA', value: 'aprobare_ca' },
  { label: 'Inregistrare', value: 'inregistrare' },
  { label: 'Publicare', value: 'publicare' },
];

const REGULATION_WORKFLOW_STATUS_OPTIONS: EducationOption[] = [
  { label: 'In asteptare', value: 'pending' },
  { label: 'Activ', value: 'active' },
  { label: 'Finalizat', value: 'completed' },
  { label: 'Returnat', value: 'returned' },
  { label: 'Anulat', value: 'cancelled' },
];

const PERSONNEL_EMPLOYMENT_OPTIONS: EducationOption[] = [
  { label: 'Titular', value: 'titular' },
  { label: 'Suplinitor', value: 'suplinitor' },
  { label: 'Plata cu ora', value: 'plata_cu_ora' },
  { label: 'Auxiliar', value: 'auxiliar' },
];

const PERSONNEL_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Activ', value: 'active' },
  { label: 'In concediu', value: 'on_leave' },
  { label: 'Vacant', value: 'vacant' },
  { label: 'Inactiv', value: 'inactive' },
];

const PERSONNEL_EVALUATION_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'In evaluare', value: 'in_review' },
  { label: 'Finalizata', value: 'finalized' },
];

const PERSONNEL_MOBILITY_STAGE_OPTIONS: EducationOption[] = [
  { label: 'Fara mobilitate', value: 'none' },
  { label: 'Transfer', value: 'transfer' },
  { label: 'Detasare', value: 'detasare' },
  { label: 'Restrangere', value: 'restrangere' },
];

const PERSONNEL_FILE_CATEGORY_OPTIONS: EducationOption[] = [
  { label: 'Identificare', value: 'identificare' },
  { label: 'Studii', value: 'studii' },
  { label: 'Cariera', value: 'cariera' },
  { label: 'Evaluare', value: 'evaluare' },
  { label: 'Declaratie', value: 'declaratie' },
  { label: 'Medical', value: 'medical' },
  { label: 'Disciplina', value: 'disciplina' },
  { label: 'Management', value: 'management' },
];

const PERSONNEL_FILE_SCOPE_OPTIONS: EducationOption[] = [
  { label: 'Dosar personal', value: 'dosar_personal' },
  { label: 'Dosar director', value: 'dosar_director' },
  { label: 'Dosar director adjunct', value: 'dosar_director_adjunct' },
];

const PERSONNEL_FILE_CONFIDENTIALITY_OPTIONS: EducationOption[] = [
  { label: 'Intern', value: 'intern' },
  { label: 'Confidential', value: 'confidential' },
  { label: 'Strict confidential', value: 'strict_confidential' },
];

const PERSONNEL_ACCESS_EVENT_OPTIONS: EducationOption[] = [
  { label: 'Consultare', value: 'consultare' },
  { label: 'Predare', value: 'predare' },
  { label: 'Actualizare', value: 'actualizare' },
  { label: 'Arhivare', value: 'arhivare' },
  { label: 'Export', value: 'export' },
];

const PERSONNEL_ACCESS_CHANNEL_OPTIONS: EducationOption[] = [
  { label: 'Fizic', value: 'fizic' },
  { label: 'Digital', value: 'digital' },
  { label: 'Mixt', value: 'mixt' },
];

const PERSONNEL_ASSIGNMENT_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Diriginte', value: 'diriginte' },
  { label: 'Coordonator proiect', value: 'coordonator_proiect' },
  { label: 'Responsabil comisie', value: 'responsabil_comisie' },
  { label: 'Mentor', value: 'mentor' },
  { label: 'Membru comisie', value: 'membru_comisie' },
  { label: 'Administrator structura', value: 'administrator_structura' },
];

const PERSONNEL_ASSIGNMENT_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Propus', value: 'propus' },
  { label: 'Activ', value: 'activ' },
  { label: 'Suspendat', value: 'suspendat' },
  { label: 'Incetat', value: 'incetat' },
];

const PERSONNEL_DISCIPLINARY_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Sesizare', value: 'sesizare' },
  { label: 'Cercetare', value: 'cercetare' },
  { label: 'Sanctiune', value: 'sanctiune' },
  { label: 'Contestatie', value: 'contestatie' },
];

const PERSONNEL_DISCIPLINARY_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Deschis', value: 'deschis' },
  { label: 'In cercetare', value: 'in_cercetare' },
  { label: 'Solutionat', value: 'solutionat' },
  { label: 'Contestat', value: 'contestat' },
  { label: 'Inchis', value: 'inchis' },
];

const EVALUATION_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'Depusa', value: 'submitted' },
  { label: 'Evaluata', value: 'reviewed' },
  { label: 'Aprobata', value: 'approved' },
  { label: 'Contestata', value: 'contested' },
];

const EVALUATION_QUALIFICATION_OPTIONS: EducationOption[] = [
  { label: 'Foarte bine', value: 'foarte_bine' },
  { label: 'Bine', value: 'bine' },
  { label: 'Satisfacator', value: 'satisfacator' },
  { label: 'Nesatisfacator', value: 'nesatisfacator' },
];

const EVALUATION_APPEAL_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Depusa', value: 'submitted' },
  { label: 'In analiza', value: 'review' },
  { label: 'Admisa', value: 'accepted' },
  { label: 'Respinsa', value: 'rejected' },
  { label: 'Solutionata', value: 'resolved' },
];

const EVALUATION_RESULT_DOCUMENT_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Fisa evaluare', value: 'fisa_evaluare' },
  { label: 'Comunicare', value: 'comunicare' },
  { label: 'Decizie', value: 'decizie' },
  { label: 'Raport final', value: 'raport_final' },
];

const EVALUATION_SELF_REVIEW_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Autoevaluare', value: 'autoevaluare' },
  { label: 'Performanta', value: 'performanta' },
  { label: 'Dezvoltare', value: 'dezvoltare' },
  { label: 'Impact', value: 'impact' },
];

const EVALUATION_SELF_REVIEW_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'Depusa', value: 'submitted' },
  { label: 'Validata', value: 'validated' },
  { label: 'Returnata', value: 'returned' },
];

const EVALUATION_CRITERION_CATEGORY_OPTIONS: EducationOption[] = [
  { label: 'Proiectare', value: 'proiectare' },
  { label: 'Predare', value: 'predare' },
  { label: 'Evaluare', value: 'evaluare' },
  { label: 'Management clasa', value: 'management_clasa' },
  { label: 'Dezvoltare', value: 'dezvoltare' },
  { label: 'Parteneriat', value: 'parteneriat' },
];

const EVALUATION_CRITERION_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'Revizuit', value: 'reviewed' },
  { label: 'Validat', value: 'validated' },
  { label: 'Contestat', value: 'contested' },
];

const DECLARATION_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Declaratie de interese', value: 'interests' },
  { label: 'Declaratie de avere', value: 'assets' },
  { label: 'Declaratie GDPR', value: 'gdpr' },
  { label: 'Declaratie de autenticitate', value: 'authenticity' },
];

const DECLARATION_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'Depusa', value: 'submitted' },
  { label: 'Validata', value: 'validated' },
  { label: 'Expirata', value: 'expired' },
];

const MOBILITY_REQUEST_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Transfer', value: 'transfer' },
  { label: 'Detasare', value: 'detasare' },
  { label: 'Pretransfer', value: 'pretransfer' },
  { label: 'Restrangere', value: 'restrangere' },
];

const MOBILITY_STAGE_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'Depus', value: 'submitted' },
  { label: 'In analiza', value: 'review' },
  { label: 'Aprobat', value: 'approved' },
  { label: 'Finalizat', value: 'completed' },
];

const MOBILITY_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Deschis', value: 'open' },
  { label: 'In asteptare', value: 'pending' },
  { label: 'Aprobat', value: 'approved' },
  { label: 'Respins', value: 'rejected' },
  { label: 'Finalizat', value: 'completed' },
];

const MOBILITY_DOCUMENT_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Cerere', value: 'cerere' },
  { label: 'Adeverinta', value: 'adeverinta' },
  { label: 'Aviz', value: 'aviz' },
  { label: 'Fisa evaluare', value: 'fisa_evaluare' },
  { label: 'Decizie', value: 'decizie' },
  { label: 'Anexa', value: 'anexa' },
];

const MOBILITY_STAGE_SCOPE_OPTIONS: EducationOption[] = [
  { label: 'Depunere', value: 'depunere' },
  { label: 'Verificare', value: 'verificare' },
  { label: 'Sedinta', value: 'sedinta' },
  { label: 'Aprobare', value: 'aprobare' },
  { label: 'Emitere', value: 'emitere' },
];

const DOSSIER_VALIDATION_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'Depus', value: 'submitted' },
  { label: 'Validat', value: 'validated' },
  { label: 'Respins', value: 'rejected' },
];

const MOBILITY_CRITERION_CATEGORY_OPTIONS: EducationOption[] = [
  { label: 'Studii', value: 'studii' },
  { label: 'Vechime', value: 'vechime' },
  { label: 'Performanta', value: 'performanta' },
  { label: 'Social', value: 'social' },
  { label: 'Administrativ', value: 'administrativ' },
];

const APPEAL_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Depusa', value: 'submitted' },
  { label: 'In analiza', value: 'review' },
  { label: 'Admisa', value: 'accepted' },
  { label: 'Respinsa', value: 'rejected' },
  { label: 'Solutionata', value: 'resolved' },
];

const MOBILITY_FINAL_DECISION_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Validare dosar', value: 'validare_dosar' },
  { label: 'Repartizare', value: 'repartizare' },
  { label: 'Transfer', value: 'transfer' },
  { label: 'Detasare', value: 'detasare' },
  { label: 'Solutionare contestatie', value: 'solutionare_contestatie' },
];

const MOBILITY_FINAL_DECISION_OUTCOME_OPTIONS: EducationOption[] = [
  { label: 'Admis', value: 'admis' },
  { label: 'Respins', value: 'respins' },
  { label: 'Redistribuit', value: 'redistribuit' },
  { label: 'Rezerva', value: 'rezerva' },
];

const RESULT_DOCUMENT_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Decizie', value: 'decizie' },
  { label: 'Comunicare', value: 'comunicare' },
  { label: 'Adeverinta', value: 'adeverinta' },
  { label: 'Raport final', value: 'raport_final' },
];

const RESULT_DELIVERY_CHANNEL_OPTIONS: EducationOption[] = [
  { label: 'Registratura', value: 'registratura' },
  { label: 'Email', value: 'email' },
  { label: 'Circuit intern', value: 'intern' },
  { label: 'Posta', value: 'posta' },
];

const RESULT_DELIVERY_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Pregatit', value: 'pregatit' },
  { label: 'Emis', value: 'emis' },
  { label: 'Transmis', value: 'transmis' },
  { label: 'Confirmat', value: 'confirmat' },
];

const MERIT_CATEGORY_OPTIONS: EducationOption[] = [
  { label: 'Predare', value: 'predare' },
  { label: 'Management', value: 'management' },
  { label: 'Consiliere', value: 'consiliere' },
  { label: 'Auxiliar', value: 'auxiliar' },
];

const MERIT_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'Depus', value: 'submitted' },
  { label: 'Evaluat', value: 'evaluated' },
  { label: 'Aprobat', value: 'approved' },
  { label: 'Finantat', value: 'funded' },
];

const MERIT_DOCUMENT_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Cerere', value: 'cerere' },
  { label: 'Declaratie', value: 'declaratie' },
  { label: 'Autoevaluare', value: 'autoevaluare' },
  { label: 'Adeverinta', value: 'adeverinta' },
  { label: 'Portofoliu', value: 'portofoliu' },
  { label: 'Anexa', value: 'anexa' },
];

const MERIT_CRITERION_CATEGORY_OPTIONS: EducationOption[] = [
  { label: 'Performanta', value: 'performanta' },
  { label: 'Impact', value: 'impact' },
  { label: 'Dezvoltare', value: 'dezvoltare' },
  { label: 'Management', value: 'management' },
  { label: 'Incluziune', value: 'incluziune' },
];

const MERIT_PANEL_STAGE_OPTIONS: EducationOption[] = [
  { label: 'Autoevaluare', value: 'autoevaluare' },
  { label: 'Evaluare comisie', value: 'evaluare_comisie' },
  { label: 'Validare finala', value: 'validare_finala' },
];

const MERIT_FINAL_DECISION_STAGE_OPTIONS: EducationOption[] = [
  { label: 'Evaluare initiala', value: 'evaluare_initiala' },
  { label: 'Solutionare contestatie', value: 'solutionare_contestatie' },
  { label: 'Validare finala', value: 'validare_finala' },
  { label: 'Finantare', value: 'finantare' },
];

const MERIT_FINAL_DECISION_OUTCOME_OPTIONS: EducationOption[] = [
  { label: 'Admis', value: 'admis' },
  { label: 'Respins', value: 'respins' },
  { label: 'Rezerva', value: 'rezerva' },
  { label: 'Finantat', value: 'finantat' },
];

const MERIT_RESULT_DOCUMENT_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Decizie', value: 'decizie' },
  { label: 'Comunicare', value: 'comunicare' },
  { label: 'Extras punctaj', value: 'extras_punctaj' },
  { label: 'Adeverinta', value: 'adeverinta' },
];

const PORTFOLIO_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'Depus', value: 'submitted' },
  { label: 'Validat', value: 'validated' },
  { label: 'Transferat', value: 'transferred' },
  { label: 'Arhivat', value: 'archived' },
];

const PORTFOLIO_TRANSFER_OPTIONS: EducationOption[] = [
  { label: 'Niciun transfer', value: 'none' },
  { label: 'Pregatit', value: 'prepared' },
  { label: 'Trimis', value: 'sent' },
  { label: 'Receptionat', value: 'received' },
];

const MEETING_MEMBER_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Presedinte', value: 'presedinte' },
  { label: 'Secretar', value: 'secretar' },
  { label: 'Membru', value: 'membru' },
  { label: 'Invitat', value: 'invitat' },
  { label: 'Observator', value: 'observator' },
];

const MEETING_ATTENDANCE_OPTIONS: EducationOption[] = [
  { label: 'Invitat', value: 'invitat' },
  { label: 'Prezent', value: 'prezent' },
  { label: 'Absent motivat', value: 'absent_motivat' },
  { label: 'Absent nemotivat', value: 'absent_nemotivat' },
];

const MEETING_DOCUMENT_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Convocator', value: 'convocator' },
  { label: 'Convocator CA', value: 'convocator_ca' },
  { label: 'Convocator CP', value: 'convocator_cp' },
  { label: 'Ordine de zi', value: 'ordine_de_zi' },
  { label: 'Lista prezenta', value: 'prezenta' },
  { label: 'Proces-verbal', value: 'proces_verbal' },
  { label: 'Proces-verbal CA', value: 'proces_verbal_ca' },
  { label: 'Proces-verbal CP', value: 'proces_verbal_cp' },
  { label: 'Registru CA', value: 'registru_ca' },
  { label: 'Registru CP', value: 'registru_cp' },
  { label: 'Decizie numire secretar CP', value: 'numire_secretar_cp' },
  { label: 'Anexa', value: 'anexa' },
  { label: 'Hotarare', value: 'hotarare' },
  { label: 'Material sedinta', value: 'material_sedinta' },
  { label: 'Delegare', value: 'delegare' },
];

const MEETING_DOCUMENT_PUBLICATION_OPTIONS: EducationOption[] = [
  { label: 'Intern', value: 'intern' },
  { label: 'Necesita anonimizare', value: 'anonimizare_necesara' },
  { label: 'Publicat', value: 'publicat' },
];

const PORTFOLIO_DOCUMENT_SOURCE_OPTIONS: EducationOption[] = [
  { label: 'Portofoliu profesional', value: 'portofoliu' },
  { label: 'Dosar personal', value: 'dosar_personal' },
];

const PORTFOLIO_DOCUMENT_EVIDENCE_OPTIONS: EducationOption[] = [
  { label: 'CV', value: 'cv' },
  { label: 'Document studii', value: 'document_studii' },
  { label: 'Document cariera', value: 'document_cariera' },
  { label: 'Planificare', value: 'planificare' },
  { label: 'Evaluare', value: 'evaluare' },
  { label: 'Formare', value: 'formare' },
  { label: 'Declaratie', value: 'declaratie' },
  { label: 'Transfer', value: 'transfer' },
  { label: 'Alt document', value: 'altele' },
];

const PORTFOLIO_DOCUMENT_AUTHENTICITY_OPTIONS: EducationOption[] = [
  { label: 'Declarat', value: 'declarat' },
  { label: 'Verificat', value: 'verificat' },
  { label: 'Respins', value: 'respins' },
];

const MEETING_VOTE_DECISION_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Hotarare', value: 'hotarare' },
  { label: 'Aviz', value: 'aviz' },
  { label: 'Informare', value: 'informare' },
  { label: 'Delegare', value: 'delegare' },
  { label: 'Aprobare', value: 'aprobare' },
];

const MEETING_VOTE_OUTCOME_OPTIONS: EducationOption[] = [
  { label: 'Adoptat', value: 'adoptat' },
  { label: 'Respins', value: 'respins' },
  { label: 'Amanat', value: 'amanat' },
];

const PORTFOLIO_CHECKLIST_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Complet', value: 'complet' },
  { label: 'Partial', value: 'partial' },
  { label: 'Lipsa', value: 'lipsa' },
  { label: 'In verificare', value: 'in_verificare' },
];

const GOVERNANCE_MEMBERSHIP_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Activ', value: 'activ' },
  { label: 'Suspendat', value: 'suspendat' },
  { label: 'Expirat', value: 'expirat' },
];

const COMMITTEE_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Evaluare personal didactic', value: 'evaluare_personal_didactic' },
  { label: 'Permanenta', value: 'permanenta' },
  { label: 'Temporara', value: 'temporara' },
  { label: 'Curriculum', value: 'curriculum' },
  { label: 'Mentorat', value: 'mentorat' },
  { label: 'Securitate', value: 'securitate' },
  { label: 'Burse', value: 'burse' },
  { label: 'Alta', value: 'alta' },
];

const COMMITTEE_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Draft', value: 'draft' },
  { label: 'Activa', value: 'active' },
  { label: 'Finalizata', value: 'completed' },
  { label: 'Arhivata', value: 'archived' },
];

const COMMITTEE_MEMBER_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Presedinte', value: 'presedinte' },
  { label: 'Secretar', value: 'secretar' },
  { label: 'Membru', value: 'membru' },
  { label: 'Observator', value: 'observator' },
  { label: 'Invitat', value: 'invitat' },
];

const COMMITTEE_MEMBER_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Activ', value: 'active' },
  { label: 'Inactiv', value: 'inactive' },
  { label: 'Inlocuit', value: 'replaced' },
];

const GOVERNANCE_RESOLUTION_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Hotarare', value: 'hotarare' },
  { label: 'Decizie', value: 'decizie' },
  { label: 'Aviz', value: 'aviz' },
];

const GOVERNANCE_RESOLUTION_PUBLICATION_OPTIONS: EducationOption[] = [
  { label: 'Intern', value: 'intern' },
  { label: 'Pregatit pentru publicare', value: 'pregatit_publicare' },
  { label: 'Publicat', value: 'publicat' },
];

const GOVERNANCE_RESOLUTION_ANONYMIZATION_OPTIONS: EducationOption[] = [
  { label: 'Necesara', value: 'necesara' },
  { label: 'Finalizata', value: 'finalizata' },
  { label: 'Nu este necesara', value: 'nu_este_necesara' },
];

const PORTFOLIO_TRANSFER_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Predare', value: 'predare' },
  { label: 'Primire', value: 'primire' },
  { label: 'Mutare', value: 'mutare' },
  { label: 'Detasare', value: 'detasare' },
];

const PORTFOLIO_TRANSFER_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Pregatit', value: 'pregatit' },
  { label: 'Trimis', value: 'trimis' },
  { label: 'Receptionat', value: 'receptionat' },
  { label: 'Inchis', value: 'inchis' },
];

const PORTFOLIO_REVIEW_STAGE_OPTIONS: EducationOption[] = [
  { label: 'Depunere', value: 'depunere' },
  { label: 'Verificare secretariat', value: 'verificare_secretariat' },
  { label: 'Validare manageriala', value: 'validare_manageriala' },
  { label: 'Reverificare', value: 'reverificare' },
];

const PORTFOLIO_REVIEW_OUTCOME_OPTIONS: EducationOption[] = [
  { label: 'Acceptat', value: 'acceptat' },
  { label: 'Completari', value: 'completari' },
  { label: 'Respins', value: 'respins' },
];

const PORTFOLIO_VALORIFICATION_SCOPE_OPTIONS: EducationOption[] = [
  { label: 'Licentiere', value: 'licentiere' },
  { label: 'Debut', value: 'debut' },
  { label: 'Definitivat', value: 'definitivat' },
  { label: 'Grad didactic II', value: 'grad_ii' },
  { label: 'Grad didactic I', value: 'grad_i' },
  { label: 'Evaluare profesionala', value: 'evaluare_profesionala' },
  { label: 'Mobilitate', value: 'mobilitate' },
  { label: 'Dezvoltare profesionala', value: 'dezvoltare_profesionala' },
  { label: 'Inspectie scolara', value: 'inspectie_scolara' },
  { label: 'Evaluare externa a calitatii', value: 'evaluare_externa_calitate' },
  { label: 'Gradatie de merit', value: 'gradatie_merit' },
  { label: 'Distinctie / premiu', value: 'distinctie_premiu' },
];

const PORTFOLIO_VALORIFICATION_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Planificat', value: 'planificat' },
  { label: 'In pregatire', value: 'in_pregatire' },
  { label: 'Transmis', value: 'transmis' },
  { label: 'Validat', value: 'validat' },
  { label: 'Finalizat', value: 'finalizat' },
];

const MEETING_MINUTE_FOLLOW_UP_OPTIONS: EducationOption[] = [
  { label: 'De stabilit', value: 'de_stabilit' },
  { label: 'In urmarire', value: 'in_urmarire' },
  { label: 'Realizat', value: 'realizat' },
  { label: 'Amanat', value: 'amanat' },
  { label: 'Inchis', value: 'inchis' },
];

const PORTFOLIO_CUSTODY_EVENT_OPTIONS: EducationOption[] = [
  { label: 'Preluare', value: 'preluare' },
  { label: 'Consultare', value: 'consultare' },
  { label: 'Transfer', value: 'transfer' },
  { label: 'Arhivare', value: 'arhivare' },
  { label: 'Restituire', value: 'restituire' },
];

const PORTFOLIO_CUSTODY_ACCESS_MODE_OPTIONS: EducationOption[] = [
  { label: 'Fizic', value: 'fizic' },
  { label: 'Digital', value: 'digital' },
  { label: 'Mixt', value: 'mixt' },
];

const PUBLICATION_DOMAIN_OPTIONS: EducationOption[] = [
  { label: 'Guvernanta', value: 'guvernanta' },
  { label: 'Documente manageriale', value: 'documente_manageriale' },
  { label: 'Portofolii', value: 'portofolii' },
  { label: 'Regulamente', value: 'regulamente' },
  { label: 'Conformitate', value: 'conformitate' },
];

const PUBLICATION_ENTITY_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Hotarare', value: 'hotarare' },
  { label: 'Proces-verbal', value: 'proces_verbal' },
  { label: 'Procedura portofoliu', value: 'procedura_portofoliu' },
  { label: 'ROF', value: 'rof' },
  { label: 'ROI', value: 'roi' },
  { label: 'PDI / PAS', value: 'pdi_pas' },
  { label: 'Raport', value: 'raport' },
  { label: 'Anunt', value: 'anunt' },
];

const PUBLICATION_CHANNEL_OPTIONS: EducationOption[] = [
  { label: 'Site public', value: 'site_public' },
  { label: 'Avizier', value: 'avizier' },
  { label: 'Intranet', value: 'intranet' },
  { label: 'Registratura', value: 'registratura' },
];

const PUBLICATION_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Pregatit', value: 'pregatit' },
  { label: 'Publicat', value: 'publicat' },
  { label: 'Retras', value: 'retras' },
];

const DECISION_ISSUANCE_TYPE_OPTIONS: EducationOption[] = [
  { label: 'Decizie', value: 'decizie' },
  { label: 'Extras', value: 'extras' },
  { label: 'Comunicare', value: 'comunicare' },
  { label: 'Adeverinta', value: 'adeverinta' },
  { label: 'Dispozitie', value: 'dispozitie' },
];

const DECISION_ISSUANCE_CHANNEL_OPTIONS: EducationOption[] = [
  { label: 'Circuit intern', value: 'intern' },
  { label: 'Email', value: 'email' },
  { label: 'Registratura', value: 'registratura' },
  { label: 'Avizier', value: 'avizier' },
  { label: 'Site', value: 'site' },
];

const DECISION_ISSUANCE_STATUS_OPTIONS: EducationOption[] = [
  { label: 'Proiect', value: 'draft' },
  { label: 'Semnat', value: 'semnat' },
  { label: 'Transmis', value: 'transmis' },
  { label: 'Confirmat', value: 'confirmat' },
  { label: 'Returnat', value: 'returnat' },
];

const DECISION_PUBLICATION_STEP_OPTIONS: EducationOption[] = [
  { label: 'Analiza juridica', value: 'analiza_juridica' },
  { label: 'Anonimizare', value: 'anonimizare' },
  { label: 'Aprobare publicare', value: 'aprobare_publicare' },
  { label: 'Publicare', value: 'publicare' },
  { label: 'Retragere', value: 'retragere' },
];

const DECISION_PUBLICATION_STEP_STATUS_OPTIONS: EducationOption[] = [
  { label: 'In asteptare', value: 'pending' },
  { label: 'In lucru', value: 'in_progress' },
  { label: 'Finalizat', value: 'completed' },
  { label: 'Blocat', value: 'blocked' },
];

const MEETING_PARTICIPANTS_CHILD: EducationDetailChildResourceConfig = {
  key: 'meeting_participants',
  label: 'Participanti sedinta',
  icon: 'pi pi-users',
  description: 'Prezenta nominala, rol, drept de vot si semnatura pentru sedinta selectata.',
  listEndpoint: (parentRow) => `/api/education/governance/meetings/${parentRow['id']}/participants`,
  detailEndpoint: (parentRow, childRow) => `/api/education/governance/meetings/${parentRow['id']}/participants/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/governance/meetings/${parentRow['id']}/participants`,
  readPermission: 'education.governance.read',
  managePermission: 'education.governance.manage',
  columns: [
    { field: 'full_name', header: 'Nume', sortable: true, filter: 'text', width: '18rem' },
    { field: 'role_name', header: 'Rol', sortable: true, filter: 'text', width: '14rem' },
    { field: 'member_type', header: 'Calitate', type: 'tag', sortable: true, filter: 'select', options: MEETING_MEMBER_TYPE_OPTIONS, width: '10rem' },
    { field: 'attendance_status', header: 'Prezenta', type: 'tag', sortable: true, filter: 'select', options: MEETING_ATTENDANCE_OPTIONS, width: '11rem' },
    { field: 'voting_right', header: 'Vot', type: 'boolean', sortable: true, width: '7rem' },
    { field: 'signature_present', header: 'Semnatura', type: 'boolean', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'full_name', label: 'Nume complet', type: 'text', wide: true, required: true },
    { field: 'role_name', label: 'Rol in sedinta', type: 'text', required: true },
    { field: 'member_type', label: 'Calitate', type: 'select', options: MEETING_MEMBER_TYPE_OPTIONS, defaultValue: 'membru', required: true },
    { field: 'attendance_status', label: 'Status prezenta', type: 'select', options: MEETING_ATTENDANCE_OPTIONS, defaultValue: 'prezent', required: true },
    { field: 'voting_right', label: 'Are drept de vot', type: 'boolean', defaultValue: true },
    { field: 'signature_present', label: 'Semnatura prezenta', type: 'boolean', defaultValue: false },
    { field: 'notes', label: 'Observatii', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista participanti pentru sedinta selectata.',
};

const MEETING_DOCUMENTS_CHILD: EducationDetailChildResourceConfig = {
  key: 'meeting_documents',
  label: 'Documente sedinta',
  icon: 'pi pi-file',
  description: 'Convocator, ordine de zi, prezenta, proces-verbal si alte documente aferente sedintei.',
  listEndpoint: (parentRow) => `/api/education/governance/meetings/${parentRow['id']}/documents`,
  detailEndpoint: (parentRow, childRow) => `/api/education/governance/meetings/${parentRow['id']}/documents/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/governance/meetings/${parentRow['id']}/documents`,
  pdfEndpoint: (parentRow, childRow) => `/api/education/governance/meetings/${parentRow['id']}/documents/${childRow['id']}/pdf`,
  pdfFilename: (_parentRow, childRow) => `document-sedinta-${String(childRow['document_number'] ?? childRow['id'] ?? 'sedinta')}.pdf`,
  pdfActionLabel: 'Document PDF',
  readPermission: 'education.governance.read',
  managePermission: 'education.governance.manage',
  columns: [
    { field: 'document_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: MEETING_DOCUMENT_TYPE_OPTIONS, width: '12rem' },
    { field: 'title', header: 'Titlu', sortable: true, filter: 'text', width: '20rem' },
    { field: 'document_number', header: 'Nr. document', sortable: true, filter: 'text', width: '12rem' },
    { field: 'registry_number', header: 'Nr. registratura', sortable: true, filter: 'text', width: '12rem' },
    { field: 'publication_status', header: 'Publicare', type: 'tag', sortable: true, filter: 'select', options: MEETING_DOCUMENT_PUBLICATION_OPTIONS, width: '12rem' },
    { field: 'issued_on', header: 'Data', sortable: true, filter: 'text', width: '9rem' },
  ],
  createFields: [
    { field: 'document_type', label: 'Tip document', type: 'select', options: MEETING_DOCUMENT_TYPE_OPTIONS, defaultValue: 'convocator', required: true },
    { field: 'title', label: 'Titlu', type: 'text', wide: true, required: true },
    { field: 'document_number', label: 'Numar document', type: 'text' },
    { field: 'registry_number', label: 'Numar registratura', type: 'text' },
    { field: 'publication_status', label: 'Status publicare', type: 'select', options: MEETING_DOCUMENT_PUBLICATION_OPTIONS, defaultValue: 'intern', required: true },
    { field: 'custody_owner', label: 'Custode', type: 'text' },
    { field: 'signed_by', label: 'Semnat de', type: 'text' },
    { field: 'issued_on', label: 'Data emiterii', type: 'date', required: true },
    { field: 'summary', label: 'Rezumat', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista documente pentru sedinta selectata.',
};

const MEETING_VOTES_CHILD: EducationDetailChildResourceConfig = {
  key: 'meeting_votes',
  label: 'Voturi si rezultate',
  icon: 'pi pi-check-square',
  description: 'Puncte de pe ordinea de zi, rezultat de vot, temei legal si masuri de urmarire.',
  listEndpoint: (parentRow) => `/api/education/governance/meetings/${parentRow['id']}/votes`,
  detailEndpoint: (parentRow, childRow) => `/api/education/governance/meetings/${parentRow['id']}/votes/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/governance/meetings/${parentRow['id']}/votes`,
  createWizardRoute: (parentRow) => `/education/governance/votes-wizard?meetingId=${encodeURIComponent(String(parentRow['id'] ?? ''))}`,
  readPermission: 'education.governance.read',
  managePermission: 'education.governance.manage',
  columns: [
    { field: 'agenda_order', header: 'Ordine', type: 'number', sortable: true, filter: 'text', width: '7rem' },
    { field: 'subject_title', header: 'Subiect', sortable: true, filter: 'text', width: '20rem' },
    { field: 'decision_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: MEETING_VOTE_DECISION_TYPE_OPTIONS, width: '10rem' },
    { field: 'outcome', header: 'Rezultat', type: 'tag', sortable: true, filter: 'select', options: MEETING_VOTE_OUTCOME_OPTIONS, width: '10rem' },
    { field: 'votes_for', header: 'Pentru', type: 'number', sortable: true, width: '7rem' },
    { field: 'votes_against', header: 'Impotriva', type: 'number', sortable: true, width: '8rem' },
    { field: 'abstentions', header: 'Abtineri', type: 'number', sortable: true, width: '7rem' },
    { field: 'requires_follow_up', header: 'Urmarire', type: 'boolean', sortable: true, width: '7rem' },
  ],
  createFields: [
    { field: 'agenda_order', label: 'Ordine pe agenda', type: 'number', defaultValue: 1, required: true },
    { field: 'subject_title', label: 'Subiect', type: 'text', wide: true, required: true },
    { field: 'decision_type', label: 'Tip decizie', type: 'select', options: MEETING_VOTE_DECISION_TYPE_OPTIONS, defaultValue: 'hotarare', required: true },
    { field: 'outcome', label: 'Rezultat', type: 'select', options: MEETING_VOTE_OUTCOME_OPTIONS, defaultValue: 'adoptat', required: true },
    { field: 'votes_for', label: 'Voturi pentru', type: 'number', defaultValue: 0 },
    { field: 'votes_against', label: 'Voturi impotriva', type: 'number', defaultValue: 0 },
    { field: 'abstentions', label: 'Abtineri', type: 'number', defaultValue: 0 },
    { field: 'requires_follow_up', label: 'Necesita urmarire', type: 'boolean', defaultValue: false },
    { field: 'legal_basis', label: 'Temei legal', type: 'textarea', wide: true },
    { field: 'notes', label: 'Masuri / note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista voturi inregistrate pentru sedinta selectata.',
};

const MEETING_RESOLUTIONS_CHILD: EducationDetailChildResourceConfig = {
  key: 'meeting_resolutions',
  label: 'Hotarari si avize',
  icon: 'pi pi-verified',
  description: 'Acte rezultate din voturile sedintei, cu publicare si anonimizare controlata.',
  listEndpoint: (parentRow) => `/api/education/governance/meetings/${parentRow['id']}/resolutions`,
  detailEndpoint: (parentRow, childRow) => `/api/education/governance/meetings/${parentRow['id']}/resolutions/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/governance/meetings/${parentRow['id']}/resolutions`,
  createWizardRoute: (parentRow) => `/education/governance/resolutions-wizard?meetingId=${encodeURIComponent(String(parentRow['id'] ?? ''))}`,
  readPermission: 'education.governance.read',
  managePermission: 'education.governance.manage',
  pdfEndpoint: (parentRow, childRow) => `/api/education/governance/meetings/${parentRow['id']}/resolutions/${childRow['id']}/pdf`,
  pdfFilename: (_parentRow, childRow) => `hotarare-${String(childRow['resolution_code'] ?? childRow['id'] ?? 'sedinta')}.pdf`,
  pdfActionLabel: 'Hotarare PDF',
  columns: [
    { field: 'resolution_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'title', header: 'Titlu', sortable: true, filter: 'text', width: '22rem' },
    { field: 'resolution_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: GOVERNANCE_RESOLUTION_TYPE_OPTIONS, width: '10rem' },
    { field: 'publication_status', header: 'Publicare', type: 'tag', sortable: true, filter: 'select', options: GOVERNANCE_RESOLUTION_PUBLICATION_OPTIONS, width: '14rem' },
    { field: 'anonymization_state', header: 'Anonimizare', type: 'tag', sortable: true, filter: 'select', options: GOVERNANCE_RESOLUTION_ANONYMIZATION_OPTIONS, width: '13rem' },
    { field: 'issued_on', header: 'Data', sortable: true, width: '9rem' },
    { field: 'signed_by', header: 'Semnat de', sortable: true, filter: 'text', width: '14rem' },
  ],
  createFields: [
    {
      field: 'vote_id',
      label: 'Subiect vot',
      type: 'select',
      required: true,
      options: ({ resource, parentRow, childRows }) => {
        if (!parentRow) {
          return [];
        }
        const voteChild = resource.detailChildren?.find((child) => child.key === 'meeting_votes');
        if (!voteChild) {
          return [];
        }
        return childRows(voteChild, parentRow).map((row) => ({
          label: `${String(row['agenda_order'] ?? '-')}. ${String(row['subject_title'] ?? 'Subiect fara titlu')} (${String(row['outcome'] ?? 'fara rezultat')})`,
          value: String(row['id'] ?? ''),
        }));
      },
    },
    { field: 'title', label: 'Titlu hotarare / aviz', type: 'text', wide: true, required: true },
    { field: 'resolution_type', label: 'Tip act', type: 'select', options: GOVERNANCE_RESOLUTION_TYPE_OPTIONS, defaultValue: 'hotarare', required: true },
    { field: 'publication_status', label: 'Status publicare', type: 'select', options: GOVERNANCE_RESOLUTION_PUBLICATION_OPTIONS, defaultValue: 'intern', required: true },
    { field: 'anonymization_state', label: 'Status anonimizare', type: 'select', options: GOVERNANCE_RESOLUTION_ANONYMIZATION_OPTIONS, defaultValue: 'necesara', required: true },
    { field: 'issued_on', label: 'Data emiterii', type: 'date', required: true },
    { field: 'signed_by', label: 'Semnat de', type: 'text' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista hotarari sau avize generate pentru sedinta selectata.',
};

const MEETING_MINUTES_CHILD: EducationDetailChildResourceConfig = {
  key: 'meeting_minutes',
  label: 'Proces-verbal structurat',
  icon: 'pi pi-file-edit',
  description: 'Puncte consemnate in procesul-verbal: dezbateri, masuri, responsabili si urmarire.',
  listEndpoint: (parentRow) => `/api/education/governance/meetings/${parentRow['id']}/minutes`,
  detailEndpoint: (parentRow, childRow) => `/api/education/governance/meetings/${parentRow['id']}/minutes/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/governance/meetings/${parentRow['id']}/minutes`,
  createWizardRoute: (parentRow) => `/education/governance/minutes-wizard?meetingId=${encodeURIComponent(String(parentRow['id'] ?? ''))}`,
  readPermission: 'education.governance.read',
  managePermission: 'education.governance.manage',
  pdfEndpoint: (parentRow, childRow) => `/api/education/governance/meetings/${parentRow['id']}/minutes/${childRow['id']}/pdf`,
  pdfFilename: (_parentRow, childRow) => `proces-verbal-${String(childRow['agenda_order'] ?? 'punct')}-${String(childRow['id'] ?? 'sedinta')}.pdf`,
  pdfActionLabel: 'Proces-verbal PDF',
  columns: [
    { field: 'agenda_order', header: 'Ordine', type: 'number', sortable: true, filter: 'text', width: '7rem' },
    { field: 'topic_title', header: 'Subiect', sortable: true, filter: 'text', width: '20rem' },
    { field: 'responsible_party', header: 'Responsabil', sortable: true, filter: 'text', width: '14rem' },
    { field: 'due_on', header: 'Termen', sortable: true, width: '9rem' },
    { field: 'follow_up_status', header: 'Urmarire', type: 'tag', sortable: true, filter: 'select', options: MEETING_MINUTE_FOLLOW_UP_OPTIONS, width: '11rem' },
    { field: 'requires_publication', header: 'Publicare', type: 'boolean', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'agenda_order', label: 'Ordine pe agenda', type: 'number', defaultValue: 1, required: true },
    { field: 'topic_title', label: 'Subiect consemnat', type: 'text', wide: true, required: true },
    { field: 'discussion_summary', label: 'Rezumat dezbateri', type: 'textarea', wide: true, required: true },
    { field: 'decision_summary', label: 'Rezumat decizie / masura', type: 'textarea', wide: true, required: true },
    { field: 'responsible_party', label: 'Responsabil', type: 'text' },
    { field: 'due_on', label: 'Termen', type: 'date' },
    { field: 'follow_up_status', label: 'Status urmarire', type: 'select', options: MEETING_MINUTE_FOLLOW_UP_OPTIONS, defaultValue: 'de_stabilit', required: true },
    { field: 'requires_publication', label: 'Necesita publicare', type: 'boolean', defaultValue: false },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista inca elemente structurate de proces-verbal pentru sedinta selectata.',
};

const MANAGERIAL_DOCUMENTS_CHILD: EducationDetailChildResourceConfig = {
  key: 'managerial_documents',
  label: 'Documente dosar',
  icon: 'pi pi-folder',
  description: 'Documente de diagnoza, prognoza, evidenta si anexe, cu versiune, publicare si trasabilitate.',
  listEndpoint: (parentRow) => `/api/education/managerial/records/${parentRow['id']}/documents`,
  detailEndpoint: (parentRow, childRow) => `/api/education/managerial/records/${parentRow['id']}/documents/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/managerial/records/${parentRow['id']}/documents`,
  pdfEndpoint: (parentRow, childRow) => `/api/education/managerial/records/${parentRow['id']}/documents/${childRow['id']}/pdf`,
  pdfFilename: (_parentRow, childRow) => `document-managerial-${String(childRow['document_code'] ?? childRow['id'] ?? 'dosar')}.pdf`,
  pdfActionLabel: 'Document PDF',
  readPermission: 'education.managerial.read',
  managePermission: 'education.managerial.manage',
  columns: [
    { field: 'document_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'document_category', header: 'Categorie', type: 'tag', sortable: true, filter: 'select', options: MANAGERIAL_DOCUMENT_CATEGORY_OPTIONS, width: '12rem' },
    { field: 'title', header: 'Titlu document', sortable: true, filter: 'text', width: '22rem' },
    { field: 'document_status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: MANAGERIAL_DOCUMENT_STATUS_OPTIONS, width: '10rem' },
    { field: 'version_label', header: 'Versiune', sortable: true, filter: 'text', width: '9rem' },
    { field: 'owner_name', header: 'Responsabil', sortable: true, filter: 'text', width: '14rem' },
    { field: 'registered_on', header: 'Inregistrat', sortable: true, width: '9rem' },
    { field: 'publication_required', header: 'Publicare', type: 'boolean', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'document_category', label: 'Categorie document', type: 'select', options: MANAGERIAL_DOCUMENT_CATEGORY_OPTIONS, defaultValue: 'planificare', required: true },
    { field: 'title', label: 'Titlu document', type: 'text', wide: true, required: true },
    { field: 'document_status', label: 'Status document', type: 'select', options: MANAGERIAL_DOCUMENT_STATUS_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'version_label', label: 'Versiune', type: 'text', defaultValue: 'v1.0', required: true },
    { field: 'mandatory', label: 'Document obligatoriu', type: 'boolean', defaultValue: true },
    { field: 'publication_required', label: 'Necesita publicare', type: 'boolean', defaultValue: false },
    { field: 'registered_on', label: 'Data inregistrarii', type: 'date', required: true },
    { field: 'approved_on', label: 'Data aprobarii', type: 'date' },
    { field: 'owner_name', label: 'Responsabil', type: 'text' },
    { field: 'file_reference', label: 'Referinta registratura / fisier', type: 'text' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista documente configurate pentru dosarul managerial selectat.',
};

const MANAGERIAL_WORKFLOW_CHILD: EducationDetailChildResourceConfig = {
  key: 'managerial_workflow',
  label: 'Flux avizare',
  icon: 'pi pi-directions-alt',
  description: 'Etape de elaborare, avizare, aprobare, publicare si arhivare pentru fiecare dosar managerial.',
  listEndpoint: (parentRow) => `/api/education/managerial/records/${parentRow['id']}/workflow`,
  detailEndpoint: (parentRow, childRow) => `/api/education/managerial/records/${parentRow['id']}/workflow/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/managerial/records/${parentRow['id']}/workflow`,
  readPermission: 'education.managerial.read',
  managePermission: 'education.managerial.manage',
  columns: [
    { field: 'stage_order', header: 'Ordine', type: 'number', sortable: true, width: '7rem' },
    { field: 'stage_type', header: 'Etapa', type: 'tag', sortable: true, filter: 'select', options: MANAGERIAL_WORKFLOW_STAGE_OPTIONS, width: '13rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: MANAGERIAL_WORKFLOW_STATUS_OPTIONS, width: '10rem' },
    { field: 'assigned_to', header: 'Responsabil', sortable: true, filter: 'text', width: '16rem' },
    { field: 'due_on', header: 'Termen', sortable: true, width: '9rem' },
    { field: 'completed_on', header: 'Finalizat', sortable: true, width: '9rem' },
    { field: 'requires_signature', header: 'Semnatura', type: 'boolean', sortable: true, width: '8rem' },
    { field: 'decision_reference', header: 'Referinta', sortable: true, filter: 'text', width: '12rem' },
  ],
  createFields: [
    { field: 'stage_order', label: 'Ordine etapa', type: 'number', defaultValue: 1, required: true },
    { field: 'stage_type', label: 'Etapa flux', type: 'select', options: MANAGERIAL_WORKFLOW_STAGE_OPTIONS, defaultValue: 'elaborare', required: true },
    { field: 'status', label: 'Status etapa', type: 'select', options: MANAGERIAL_WORKFLOW_STATUS_OPTIONS, defaultValue: 'pending', required: true },
    { field: 'assigned_to', label: 'Responsabil', type: 'text', wide: true, required: true },
    { field: 'due_on', label: 'Termen etapa', type: 'date', required: true },
    { field: 'completed_on', label: 'Data finalizarii', type: 'date' },
    { field: 'requires_signature', label: 'Necesita semnatura', type: 'boolean', defaultValue: false },
    { field: 'decision_reference', label: 'Referinta PV / hotarare', type: 'text' },
    { field: 'outcome_note', label: 'Rezultat / observatii', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista etape de flux pentru dosarul managerial selectat.',
};

const REGULATION_VERSIONS_CHILD: EducationDetailChildResourceConfig = {
  key: 'regulation_versions',
  label: 'Versiuni document',
  icon: 'pi pi-history',
  description: 'Versionare ROF/ROI, cu schimbari, intrare in vigoare si trasabilitate pentru fiecare editie.',
  listEndpoint: (parentRow) => `/api/education/regulations/records/${parentRow['id']}/versions`,
  detailEndpoint: (parentRow, childRow) => `/api/education/regulations/records/${parentRow['id']}/versions/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/regulations/records/${parentRow['id']}/versions`,
  readPermission: 'education.regulations.read',
  managePermission: 'education.regulations.manage',
  columns: [
    { field: 'version_label', header: 'Versiune', sortable: true, filter: 'text', width: '9rem' },
    { field: 'version_status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: REGULATION_VERSION_STATUS_OPTIONS, width: '12rem' },
    { field: 'change_summary', header: 'Schimbari', sortable: true, filter: 'text', width: '24rem' },
    { field: 'prepared_by', header: 'Pregatit de', sortable: true, filter: 'text', width: '14rem' },
    { field: 'approved_on', header: 'Aprobat', sortable: true, width: '9rem' },
    { field: 'effective_from', header: 'In vigoare', sortable: true, width: '9rem' },
    { field: 'published_on', header: 'Publicat', sortable: true, width: '9rem' },
  ],
  createFields: [
    { field: 'version_label', label: 'Versiune', type: 'text', defaultValue: 'v1.0', required: true },
    { field: 'version_status', label: 'Status versiune', type: 'select', options: REGULATION_VERSION_STATUS_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'change_summary', label: 'Rezumat modificari', type: 'textarea', wide: true, required: true },
    { field: 'effective_from', label: 'Intra in vigoare la', type: 'date', required: true },
    { field: 'approved_on', label: 'Aprobat la', type: 'date' },
    { field: 'published_on', label: 'Publicat la', type: 'date' },
    { field: 'prepared_by', label: 'Pregatit de', type: 'text', required: true },
    { field: 'file_reference', label: 'Referinta fisier / registratura', type: 'text' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista versiuni inregistrate pentru regulamentul selectat.',
};

const REGULATION_WORKFLOW_CHILD: EducationDetailChildResourceConfig = {
  key: 'regulation_workflow',
  label: 'Consultare si aprobare',
  icon: 'pi pi-sitemap',
  description: 'Circuit complet pentru redactare, consultare publica, avizare, aprobare, inregistrare si publicare.',
  listEndpoint: (parentRow) => `/api/education/regulations/records/${parentRow['id']}/workflow`,
  detailEndpoint: (parentRow, childRow) => `/api/education/regulations/records/${parentRow['id']}/workflow/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/regulations/records/${parentRow['id']}/workflow`,
  readPermission: 'education.regulations.read',
  managePermission: 'education.regulations.manage',
  columns: [
    { field: 'phase_order', header: 'Ordine', type: 'number', sortable: true, width: '7rem' },
    { field: 'phase_type', header: 'Faza', type: 'tag', sortable: true, filter: 'select', options: REGULATION_WORKFLOW_PHASE_OPTIONS, width: '14rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: REGULATION_WORKFLOW_STATUS_OPTIONS, width: '10rem' },
    { field: 'audience', header: 'Public / organism', sortable: true, filter: 'text', width: '18rem' },
    { field: 'started_on', header: 'Start', sortable: true, width: '9rem' },
    { field: 'due_on', header: 'Termen', sortable: true, width: '9rem' },
    { field: 'completed_on', header: 'Finalizat', sortable: true, width: '9rem' },
    { field: 'feedback_count', header: 'Observatii', type: 'number', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'phase_order', label: 'Ordine faza', type: 'number', defaultValue: 1, required: true },
    { field: 'phase_type', label: 'Faza workflow', type: 'select', options: REGULATION_WORKFLOW_PHASE_OPTIONS, defaultValue: 'redactare', required: true },
    { field: 'status', label: 'Status faza', type: 'select', options: REGULATION_WORKFLOW_STATUS_OPTIONS, defaultValue: 'pending', required: true },
    { field: 'audience', label: 'Public tinta / organism', type: 'text', wide: true, required: true },
    { field: 'started_on', label: 'Data start', type: 'date', required: true },
    { field: 'due_on', label: 'Termen limita', type: 'date', required: true },
    { field: 'completed_on', label: 'Data finalizarii', type: 'date' },
    { field: 'feedback_count', label: 'Numar observatii', type: 'number', defaultValue: 0 },
    { field: 'decision_reference', label: 'Referinta decizie / anunt', type: 'text' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista faze de consultare si aprobare pentru regulamentul selectat.',
};

const PERSONNEL_FILE_DOCUMENTS_CHILD: EducationDetailChildResourceConfig = {
  key: 'personnel_file_documents',
  label: 'Dosar personal',
  icon: 'pi pi-folder-open',
  description: 'Documente distincte ale dosarului personal, separate de portofoliul profesional si de dosarele manageriale.',
  listEndpoint: (parentRow) => `/api/education/personnel/records/${parentRow['id']}/file-documents`,
  detailEndpoint: (parentRow, childRow) => `/api/education/personnel/records/${parentRow['id']}/file-documents/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/personnel/records/${parentRow['id']}/file-documents`,
  readPermission: 'education.personnel.files.read',
  managePermission: 'education.personnel.files.manage',
  columns: [
    { field: 'document_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'document_category', header: 'Categorie', type: 'tag', sortable: true, filter: 'select', options: PERSONNEL_FILE_CATEGORY_OPTIONS, width: '12rem' },
    { field: 'document_title', header: 'Document', sortable: true, filter: 'text', width: '22rem' },
    { field: 'file_scope', header: 'Dosar', type: 'tag', sortable: true, filter: 'select', options: PERSONNEL_FILE_SCOPE_OPTIONS, width: '14rem' },
    { field: 'confidentiality_level', header: 'Confidentialitate', type: 'tag', sortable: true, filter: 'select', options: PERSONNEL_FILE_CONFIDENTIALITY_OPTIONS, width: '13rem' },
    { field: 'issued_on', header: 'Emis la', sortable: true, width: '9rem' },
    { field: 'sensitive_data', header: 'Sensibil', type: 'boolean', sortable: true, width: '7rem' },
    { field: 'included_in_portfolio', header: 'In portofoliu', type: 'boolean', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'document_category', label: 'Categorie document', type: 'select', options: PERSONNEL_FILE_CATEGORY_OPTIONS, defaultValue: 'cariera', required: true },
    { field: 'document_title', label: 'Titlu document', type: 'text', wide: true, required: true },
    { field: 'file_scope', label: 'Tip dosar', type: 'select', options: PERSONNEL_FILE_SCOPE_OPTIONS, defaultValue: 'dosar_personal', required: true },
    { field: 'confidentiality_level', label: 'Nivel confidentialitate', type: 'select', options: PERSONNEL_FILE_CONFIDENTIALITY_OPTIONS, defaultValue: 'confidential', required: true },
    { field: 'issued_on', label: 'Data emiterii', type: 'date', required: true },
    { field: 'expires_on', label: 'Valabil pana la', type: 'date' },
    { field: 'file_reference', label: 'Referinta fisier / registratura', type: 'text' },
    { field: 'sensitive_data', label: 'Contine date sensibile', type: 'boolean', defaultValue: true },
    { field: 'included_in_portfolio', label: 'Poate fi referit in portofoliu', type: 'boolean', defaultValue: false },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista documente in dosarul personal pentru cadrul selectat.',
};

const PERSONNEL_ACCESS_EVENTS_CHILD: EducationDetailChildResourceConfig = {
  key: 'personnel_access_events',
  label: 'Jurnal acces',
  icon: 'pi pi-lock',
  description: 'Trasabilitate pentru consultare, actualizare, predare si export al dosarului personal.',
  listEndpoint: (parentRow) => `/api/education/personnel/records/${parentRow['id']}/access-events`,
  detailEndpoint: (parentRow, childRow) => `/api/education/personnel/records/${parentRow['id']}/access-events/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/personnel/records/${parentRow['id']}/access-events`,
  readPermission: 'education.personnel.access.read',
  managePermission: 'education.personnel.access.manage',
  columns: [
    { field: 'event_type', header: 'Eveniment', type: 'tag', sortable: true, filter: 'select', options: PERSONNEL_ACCESS_EVENT_OPTIONS, width: '11rem' },
    { field: 'actor_name', header: 'Actor', sortable: true, filter: 'text', width: '16rem' },
    { field: 'actor_role', header: 'Rol', sortable: true, filter: 'text', width: '14rem' },
    { field: 'access_channel', header: 'Canal', type: 'tag', sortable: true, filter: 'select', options: PERSONNEL_ACCESS_CHANNEL_OPTIONS, width: '10rem' },
    { field: 'accessed_on', header: 'Acces la', sortable: true, width: '9rem' },
    { field: 'closed_on', header: 'Inchis la', sortable: true, width: '9rem' },
    { field: 'sensitive_scope', header: 'Date sensibile', type: 'boolean', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'event_type', label: 'Tip eveniment', type: 'select', options: PERSONNEL_ACCESS_EVENT_OPTIONS, defaultValue: 'consultare', required: true },
    { field: 'actor_name', label: 'Nume actor', type: 'text', required: true },
    { field: 'actor_role', label: 'Rol actor', type: 'text', required: true },
    { field: 'purpose', label: 'Scop acces', type: 'textarea', wide: true, required: true },
    { field: 'access_channel', label: 'Canal acces', type: 'select', options: PERSONNEL_ACCESS_CHANNEL_OPTIONS, defaultValue: 'digital', required: true },
    { field: 'accessed_on', label: 'Data accesului', type: 'date', required: true },
    { field: 'closed_on', label: 'Data inchiderii', type: 'date' },
    { field: 'sensitive_scope', label: 'Include date sensibile', type: 'boolean', defaultValue: true },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista accesari inregistrate pentru dosarul personal selectat.',
};

const PERSONNEL_ASSIGNMENTS_CHILD: EducationDetailChildResourceConfig = {
  key: 'personnel_assignments',
  label: 'Atribuiri functionale',
  icon: 'pi pi-briefcase',
  description: 'Dirigentie, coordonari, comisii si alte responsabilitati formale.',
  listEndpoint: (parentRow) => `/api/education/personnel/records/${parentRow['id']}/assignments`,
  detailEndpoint: (parentRow, childRow) => `/api/education/personnel/records/${parentRow['id']}/assignments/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/personnel/records/${parentRow['id']}/assignments`,
  readPermission: 'education.personnel.read',
  managePermission: 'education.personnel.manage',
  columns: [
    { field: 'assignment_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'assignment_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: PERSONNEL_ASSIGNMENT_TYPE_OPTIONS, width: '13rem' },
    { field: 'assignment_title', header: 'Atribuire', sortable: true, filter: 'text', width: '20rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: PERSONNEL_ASSIGNMENT_STATUS_OPTIONS, width: '10rem' },
    { field: 'assigned_on', header: 'De la', sortable: true, width: '9rem' },
    { field: 'ended_on', header: 'Pana la', sortable: true, width: '9rem' },
    { field: 'weekly_hours', header: 'Ore/sapt.', type: 'number', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'assignment_type', label: 'Tip atribuire', type: 'select', options: PERSONNEL_ASSIGNMENT_TYPE_OPTIONS, defaultValue: 'diriginte', required: true },
    { field: 'assignment_title', label: 'Denumire', type: 'text', wide: true, required: true },
    { field: 'status', label: 'Status', type: 'select', options: PERSONNEL_ASSIGNMENT_STATUS_OPTIONS, defaultValue: 'propus', required: true },
    { field: 'assigned_on', label: 'Data inceput', type: 'date', required: true },
    { field: 'ended_on', label: 'Data incetare', type: 'date' },
    { field: 'weekly_hours', label: 'Ore pe saptamana', type: 'number', defaultValue: 0 },
    { field: 'decision_reference', label: 'Referinta decizie', type: 'text' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista atribuiri functionale pentru aceasta persoana.',
};

const PERSONNEL_DISCIPLINARY_CASES_CHILD: EducationDetailChildResourceConfig = {
  key: 'personnel_disciplinary_cases',
  label: 'Cazuri disciplinare',
  icon: 'pi pi-shield',
  description: 'Sesizari, cercetari, sanctiuni si contestatii cu trasabilitate minima.',
  listEndpoint: (parentRow) => `/api/education/personnel/records/${parentRow['id']}/disciplinary-cases`,
  detailEndpoint: (parentRow, childRow) => `/api/education/personnel/records/${parentRow['id']}/disciplinary-cases/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/personnel/records/${parentRow['id']}/disciplinary-cases`,
  readPermission: 'education.personnel.read',
  managePermission: 'education.personnel.manage',
  columns: [
    { field: 'case_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'case_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: PERSONNEL_DISCIPLINARY_TYPE_OPTIONS, width: '12rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: PERSONNEL_DISCIPLINARY_STATUS_OPTIONS, width: '11rem' },
    { field: 'reported_on', header: 'Sesizat la', sortable: true, width: '9rem' },
    { field: 'resolved_on', header: 'Solutionat la', sortable: true, width: '10rem' },
    { field: 'committee_name', header: 'Comisie', sortable: true, filter: 'text', width: '18rem' },
    { field: 'sanction', header: 'Sanctiune', sortable: true, filter: 'text', width: '16rem' },
  ],
  createFields: [
    { field: 'case_type', label: 'Tip caz', type: 'select', options: PERSONNEL_DISCIPLINARY_TYPE_OPTIONS, defaultValue: 'sesizare', required: true },
    { field: 'status', label: 'Status', type: 'select', options: PERSONNEL_DISCIPLINARY_STATUS_OPTIONS, defaultValue: 'deschis', required: true },
    { field: 'reported_on', label: 'Data sesizare', type: 'date', required: true },
    { field: 'hearing_on', label: 'Data audierii', type: 'date' },
    { field: 'resolved_on', label: 'Data solutionarii', type: 'date' },
    { field: 'committee_name', label: 'Comisie', type: 'text' },
    { field: 'sanction', label: 'Sanctiune', type: 'text' },
    { field: 'legal_basis', label: 'Temei legal', type: 'textarea', wide: true },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista cazuri disciplinare pentru aceasta persoana.',
};

const EVALUATION_SELF_REVIEWS_CHILD: EducationDetailChildResourceConfig = {
  key: 'evaluation_self_reviews',
  label: 'Autoevaluari',
  icon: 'pi pi-pencil',
  description: 'Autoevaluare structurata pe sectiuni, cu punctaj asumat si dovezi sintetice.',
  listEndpoint: (parentRow) => `/api/education/evaluations/records/${parentRow['id']}/self-reviews`,
  detailEndpoint: (parentRow, childRow) => `/api/education/evaluations/records/${parentRow['id']}/self-reviews/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/evaluations/records/${parentRow['id']}/self-reviews`,
  readPermission: 'education.evaluations.read',
  managePermission: 'education.evaluations.manage',
  columns: [
    { field: 'review_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'section_title', header: 'Sectiune', sortable: true, filter: 'text', width: '20rem' },
    { field: 'narrative_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: EVALUATION_SELF_REVIEW_TYPE_OPTIONS, width: '12rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: EVALUATION_SELF_REVIEW_STATUS_OPTIONS, width: '10rem' },
    { field: 'completed_on', header: 'Data', sortable: true, width: '9rem' },
    { field: 'assumed_score', header: 'Punctaj', type: 'number', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'section_title', label: 'Sectiune', type: 'text', wide: true, required: true },
    { field: 'narrative_type', label: 'Tip narativ', type: 'select', options: EVALUATION_SELF_REVIEW_TYPE_OPTIONS, defaultValue: 'autoevaluare', required: true },
    { field: 'status', label: 'Status', type: 'select', options: EVALUATION_SELF_REVIEW_STATUS_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'completed_on', label: 'Data completarii', type: 'date', required: true },
    { field: 'assumed_score', label: 'Punctaj asumat (0-100)', type: 'number', defaultValue: 0, min: 0, max: 100, step: 0.01 },
    { field: 'evidence_summary', label: 'Dovezi sintetice', type: 'textarea', wide: true },
    { field: 'strengths', label: 'Puncte tari', type: 'textarea', wide: true },
    { field: 'improvement_needs', label: 'Nevoi de imbunatatire', type: 'textarea', wide: true },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista autoevaluari pentru evaluarea selectata.',
};

const EVALUATION_CRITERIA_CHILD: EducationDetailChildResourceConfig = {
  key: 'evaluation_criteria',
  label: 'Criterii',
  icon: 'pi pi-list',
  description: 'Criterii detaliate de evaluare cu scor propriu, scor evaluator si punctaj final.',
  listEndpoint: (parentRow) => `/api/education/evaluations/records/${parentRow['id']}/criteria`,
  detailEndpoint: (parentRow, childRow) => `/api/education/evaluations/records/${parentRow['id']}/criteria/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/evaluations/records/${parentRow['id']}/criteria`,
  readPermission: 'education.evaluations.read',
  managePermission: 'education.evaluations.manage',
  columns: [
    { field: 'criterion_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'criterion_category', header: 'Categorie', type: 'tag', sortable: true, filter: 'select', options: EVALUATION_CRITERION_CATEGORY_OPTIONS, width: '13rem' },
    { field: 'criterion_label', header: 'Criteriu', sortable: true, filter: 'text', width: '22rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: EVALUATION_CRITERION_STATUS_OPTIONS, width: '10rem' },
    { field: 'max_score', header: 'Maxim', type: 'number', sortable: true, width: '7rem' },
    { field: 'final_score', header: 'Final', type: 'number', sortable: true, width: '7rem' },
  ],
  createFields: [
    { field: 'criterion_category', label: 'Categorie', type: 'select', options: EVALUATION_CRITERION_CATEGORY_OPTIONS, defaultValue: 'predare', required: true },
    { field: 'criterion_label', label: 'Criteriu', type: 'text', wide: true, required: true },
    { field: 'status', label: 'Status', type: 'select', options: EVALUATION_CRITERION_STATUS_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'max_score', label: 'Punctaj maxim (0-100)', type: 'number', defaultValue: 25, required: true, min: 0, max: 100, step: 0.01 },
    { field: 'self_score', label: 'Scor autoevaluare', type: 'number', defaultValue: 0, min: 0, max: 100, step: 0.01 },
    { field: 'reviewer_score', label: 'Scor evaluator', type: 'number', defaultValue: 0, min: 0, max: 100, step: 0.01 },
    { field: 'final_score', label: 'Scor final', type: 'number', defaultValue: 0, min: 0, max: 100, step: 0.01 },
    { field: 'evidence_summary', label: 'Dovezi', type: 'textarea', wide: true },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista criterii inregistrate pentru evaluarea selectata.',
};

const EVALUATION_APPEALS_CHILD: EducationDetailChildResourceConfig = {
  key: 'evaluation_appeals',
  label: 'Contestatii',
  icon: 'pi pi-exclamation-circle',
  description: 'Cereri de reanalizare, sedinte de solutionare si integrarea rezultatului in dosarul personal.',
  listEndpoint: (parentRow) => `/api/education/evaluations/records/${parentRow['id']}/appeals`,
  detailEndpoint: (parentRow, childRow) => `/api/education/evaluations/records/${parentRow['id']}/appeals/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/evaluations/records/${parentRow['id']}/appeals`,
  readPermission: 'education.evaluations.read',
  managePermission: 'education.evaluations.manage',
  pdfEndpoint: (parentRow, childRow) => `/api/education/evaluations/records/${parentRow['id']}/appeals/${childRow['id']}/pdf`,
  pdfFilename: (_parentRow, childRow) => `contestatie-${String(childRow['appeal_code'] ?? childRow['id'] ?? 'evaluare')}.pdf`,
  pdfActionLabel: 'Contestatie PDF',
  columns: [
    { field: 'appeal_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'submitted_by', header: 'Depusa de', sortable: true, filter: 'text', width: '16rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: EVALUATION_APPEAL_STATUS_OPTIONS, width: '10rem' },
    { field: 'submitted_on', header: 'Depusa la', sortable: true, width: '9rem' },
    { field: 'hearing_on', header: 'Sedinta', sortable: true, width: '9rem' },
    { field: 'resolved_on', header: 'Solutionata', sortable: true, width: '10rem' },
    { field: 'attached_to_personnel_file', header: 'In dosar', type: 'boolean', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'submitted_by', label: 'Depusa de', type: 'text', wide: true, required: true },
    { field: 'submitted_on', label: 'Data depunerii', type: 'date', required: true },
    { field: 'status', label: 'Status contestatie', type: 'select', options: EVALUATION_APPEAL_STATUS_OPTIONS, defaultValue: 'submitted', required: true },
    { field: 'grounds', label: 'Motive contestatie', type: 'textarea', wide: true, required: true },
    { field: 'hearing_on', label: 'Data sedintei', type: 'date' },
    { field: 'resolved_on', label: 'Data solutionarii', type: 'date' },
    { field: 'decision_summary', label: 'Rezumat decizie', type: 'textarea', wide: true },
    { field: 'committee_note', label: 'Nota comisie', type: 'textarea', wide: true },
    { field: 'attached_to_personnel_file', label: 'Rezultatul merge in dosarul personal', type: 'boolean', defaultValue: true },
  ],
  emptyText: 'Nu exista contestatii pentru evaluarea selectata.',
};

const EVALUATION_RESULT_ISSUES_CHILD: EducationDetailChildResourceConfig = {
  key: 'evaluation_result_issues',
  label: 'Comunicare rezultat',
  icon: 'pi pi-send',
  description: 'Transmiterea fisei de evaluare si a comunicarii oficiale a rezultatului, cu confirmare si includere in dosarul personal.',
  listEndpoint: (parentRow) => `/api/education/evaluations/records/${parentRow['id']}/result-issues`,
  detailEndpoint: (parentRow, childRow) => `/api/education/evaluations/records/${parentRow['id']}/result-issues/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/evaluations/records/${parentRow['id']}/result-issues`,
  readPermission: 'education.evaluations.read',
  managePermission: 'education.evaluations.manage',
  pdfEndpoint: (parentRow, childRow) => `/api/education/evaluations/records/${parentRow['id']}/result-issues/${childRow['id']}/pdf`,
  pdfFilename: (_parentRow, childRow) => `comunicare-rezultat-${String(childRow['issue_code'] ?? childRow['id'] ?? 'evaluare')}.pdf`,
  pdfActionLabel: 'Comunicare PDF',
  columns: [
    { field: 'issue_code', header: 'Cod', sortable: true, filter: 'text', width: '12rem' },
    { field: 'document_type', header: 'Document', type: 'tag', sortable: true, filter: 'select', options: EVALUATION_RESULT_DOCUMENT_TYPE_OPTIONS, width: '12rem' },
    { field: 'recipient_name', header: 'Destinatar', sortable: true, filter: 'text', width: '18rem' },
    { field: 'delivery_channel', header: 'Canal', type: 'tag', sortable: true, filter: 'select', options: RESULT_DELIVERY_CHANNEL_OPTIONS, width: '10rem' },
    { field: 'delivery_status', header: 'Livrare', type: 'tag', sortable: true, filter: 'select', options: RESULT_DELIVERY_STATUS_OPTIONS, width: '10rem' },
    { field: 'issued_on', header: 'Emis la', sortable: true, width: '9rem' },
    { field: 'acknowledged_on', header: 'Confirmat la', sortable: true, width: '10rem' },
    { field: 'attached_to_personnel_file', header: 'In dosar', type: 'boolean', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'document_type', label: 'Tip document', type: 'select', options: EVALUATION_RESULT_DOCUMENT_TYPE_OPTIONS, defaultValue: 'fisa_evaluare', required: true },
    { field: 'recipient_name', label: 'Destinatar', type: 'text', wide: true, required: true },
    { field: 'recipient_role', label: 'Rol destinatar', type: 'text' },
    { field: 'delivery_channel', label: 'Canal livrare', type: 'select', options: RESULT_DELIVERY_CHANNEL_OPTIONS, defaultValue: 'registratura', required: true },
    { field: 'delivery_status', label: 'Status livrare', type: 'select', options: RESULT_DELIVERY_STATUS_OPTIONS, defaultValue: 'pregatit', required: true },
    { field: 'issued_on', label: 'Data emiterii', type: 'date', required: true },
    { field: 'delivered_on', label: 'Data predarii', type: 'date' },
    { field: 'acknowledged_on', label: 'Data confirmarii', type: 'date' },
    { field: 'registry_reference', label: 'Referinta registratura', type: 'text' },
    { field: 'attached_to_personnel_file', label: 'Include in dosarul personal', type: 'boolean', defaultValue: true },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista comunicari de rezultat pentru evaluarea selectata.',
};

const MOBILITY_DOCUMENTS_CHILD: EducationDetailChildResourceConfig = {
  key: 'mobility_documents',
  label: 'Dosar mobilitate',
  icon: 'pi pi-folder',
  description: 'Documente justificative pe etape, cu validare si trasabilitate pentru dosarul de mobilitate.',
  listEndpoint: (parentRow) => `/api/education/mobility/records/${parentRow['id']}/documents`,
  detailEndpoint: (parentRow, childRow) => `/api/education/mobility/records/${parentRow['id']}/documents/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/mobility/records/${parentRow['id']}/documents`,
  readPermission: 'education.mobility.read',
  managePermission: 'education.mobility.manage',
  columns: [
    { field: 'document_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'document_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: MOBILITY_DOCUMENT_TYPE_OPTIONS, width: '11rem' },
    { field: 'stage_scope', header: 'Etapa', type: 'tag', sortable: true, filter: 'select', options: MOBILITY_STAGE_SCOPE_OPTIONS, width: '11rem' },
    { field: 'document_title', header: 'Document', sortable: true, filter: 'text', width: '22rem' },
    { field: 'validation_status', header: 'Validare', type: 'tag', sortable: true, filter: 'select', options: DOSSIER_VALIDATION_OPTIONS, width: '10rem' },
    { field: 'submitted_by', header: 'Depus de', sortable: true, filter: 'text', width: '14rem' },
    { field: 'registered_on', header: 'Inregistrat', sortable: true, width: '9rem' },
    { field: 'mandatory', header: 'Obligatoriu', type: 'boolean', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'document_type', label: 'Tip document', type: 'select', options: MOBILITY_DOCUMENT_TYPE_OPTIONS, defaultValue: 'cerere', required: true },
    { field: 'stage_scope', label: 'Etapa dosar', type: 'select', options: MOBILITY_STAGE_SCOPE_OPTIONS, defaultValue: 'depunere', required: true },
    { field: 'document_title', label: 'Titlu document', type: 'text', wide: true, required: true },
    { field: 'registered_on', label: 'Data inregistrarii', type: 'date', required: true },
    { field: 'submitted_by', label: 'Depus de', type: 'text' },
    { field: 'verified_by', label: 'Verificat de', type: 'text' },
    { field: 'validation_status', label: 'Status validare', type: 'select', options: DOSSIER_VALIDATION_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'mandatory', label: 'Document obligatoriu', type: 'boolean', defaultValue: true },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista documente in dosarul de mobilitate selectat.',
};

const MOBILITY_SCORES_CHILD: EducationDetailChildResourceConfig = {
  key: 'mobility_scores',
  label: 'Punctaj criterii',
  icon: 'pi pi-calculator',
  description: 'Punctaj detaliat pe criterii de mobilitate, cu dovezi si marcaj de contestare.',
  listEndpoint: (parentRow) => `/api/education/mobility/records/${parentRow['id']}/scores`,
  detailEndpoint: (parentRow, childRow) => `/api/education/mobility/records/${parentRow['id']}/scores/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/mobility/records/${parentRow['id']}/scores`,
  readPermission: 'education.mobility.read',
  managePermission: 'education.mobility.manage',
  columns: [
    { field: 'criterion_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
    { field: 'criterion_category', header: 'Categorie', type: 'tag', sortable: true, filter: 'select', options: MOBILITY_CRITERION_CATEGORY_OPTIONS, width: '12rem' },
    { field: 'criterion_label', header: 'Criteriu', sortable: true, filter: 'text', width: '22rem' },
    { field: 'max_score', header: 'Maxim', type: 'number', sortable: true, width: '7rem' },
    { field: 'awarded_score', header: 'Acordat', type: 'number', sortable: true, width: '7rem' },
    { field: 'validated_by', header: 'Validat de', sortable: true, filter: 'text', width: '14rem' },
    { field: 'contested', header: 'Contestat', type: 'boolean', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'criterion_code', label: 'Cod criteriu', type: 'text', required: true },
    { field: 'criterion_label', label: 'Denumire criteriu', type: 'text', wide: true, required: true },
    { field: 'criterion_category', label: 'Categorie', type: 'select', options: MOBILITY_CRITERION_CATEGORY_OPTIONS, defaultValue: 'studii', required: true },
    { field: 'max_score', label: 'Punctaj maxim', type: 'number', defaultValue: 10, required: true },
    { field: 'awarded_score', label: 'Punctaj acordat', type: 'number', defaultValue: 0, required: true },
    { field: 'evidence_reference', label: 'Referinta dovada', type: 'text' },
    { field: 'validated_by', label: 'Validat de', type: 'text' },
    { field: 'contested', label: 'Contestat', type: 'boolean', defaultValue: false },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista punctaje pe criterii pentru cazul de mobilitate selectat.',
};

const MOBILITY_APPEALS_CHILD: EducationDetailChildResourceConfig = {
  key: 'mobility_appeals',
  label: 'Contestatii',
  icon: 'pi pi-flag',
  description: 'Contestatii privind punctajul sau etapa de mobilitate, cu sedinta si solutionare.',
  listEndpoint: (parentRow) => `/api/education/mobility/records/${parentRow['id']}/appeals`,
  detailEndpoint: (parentRow, childRow) => `/api/education/mobility/records/${parentRow['id']}/appeals/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/mobility/records/${parentRow['id']}/appeals`,
  pdfEndpoint: (parentRow, childRow) => `/api/education/mobility/records/${parentRow['id']}/appeals/${childRow['id']}/pdf`,
  pdfFilename: (_parentRow, childRow) => `contestatie-mobilitate-${childRow['appeal_code'] ?? childRow['id']}.pdf`,
  pdfActionLabel: 'Contestatie PDF',
  readPermission: 'education.mobility.read',
  managePermission: 'education.mobility.manage',
  columns: [
    { field: 'appeal_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'submitted_by', header: 'Depusa de', sortable: true, filter: 'text', width: '16rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: APPEAL_STATUS_OPTIONS, width: '10rem' },
    { field: 'submitted_on', header: 'Depusa la', sortable: true, width: '9rem' },
    { field: 'hearing_on', header: 'Sedinta', sortable: true, width: '9rem' },
    { field: 'resolved_on', header: 'Solutionata', sortable: true, width: '10rem' },
  ],
  createFields: [
    { field: 'submitted_by', label: 'Depusa de', type: 'text', wide: true, required: true },
    { field: 'submitted_on', label: 'Data depunerii', type: 'date', required: true },
    { field: 'status', label: 'Status contestatie', type: 'select', options: APPEAL_STATUS_OPTIONS, defaultValue: 'submitted', required: true },
    { field: 'grounds', label: 'Motive contestatie', type: 'textarea', wide: true, required: true },
    { field: 'hearing_on', label: 'Data sedintei', type: 'date' },
    { field: 'resolved_on', label: 'Data solutionarii', type: 'date' },
    { field: 'decision_summary', label: 'Rezumat decizie', type: 'textarea', wide: true },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista contestatii pentru cazul de mobilitate selectat.',
};

const MOBILITY_FINAL_DECISIONS_CHILD: EducationDetailChildResourceConfig = {
  key: 'mobility_final_decisions',
  label: 'Decizie finala',
  icon: 'pi pi-verified',
  description: 'Hotararea finala privind mobilitatea, cu efect, baza legala si unitate de destinatie.',
  listEndpoint: (parentRow) => `/api/education/mobility/records/${parentRow['id']}/final-decisions`,
  detailEndpoint: (parentRow, childRow) => `/api/education/mobility/records/${parentRow['id']}/final-decisions/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/mobility/records/${parentRow['id']}/final-decisions`,
  pdfEndpoint: (parentRow, childRow) => `/api/education/mobility/records/${parentRow['id']}/final-decisions/${childRow['id']}/pdf`,
  pdfFilename: (_parentRow, childRow) => `decizie-mobilitate-${childRow['decision_code'] ?? childRow['id']}.pdf`,
  pdfActionLabel: 'Decizie PDF',
  readPermission: 'education.mobility.read',
  managePermission: 'education.mobility.manage',
  columns: [
    { field: 'decision_code', header: 'Cod', sortable: true, filter: 'text', width: '12rem' },
    { field: 'decision_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: MOBILITY_FINAL_DECISION_TYPE_OPTIONS, width: '14rem' },
    { field: 'outcome', header: 'Rezultat', type: 'tag', sortable: true, filter: 'select', options: MOBILITY_FINAL_DECISION_OUTCOME_OPTIONS, width: '10rem' },
    { field: 'panel_name', header: 'Comisie', sortable: true, filter: 'text', width: '18rem' },
    { field: 'destination_unit', header: 'Unitate destinatie', sortable: true, filter: 'text', width: '18rem' },
    { field: 'approved_on', header: 'Aprobata la', sortable: true, width: '9rem' },
    { field: 'effective_from', header: 'Aplicare de la', sortable: true, width: '10rem' },
  ],
  createFields: [
    { field: 'decision_type', label: 'Tip decizie', type: 'select', options: MOBILITY_FINAL_DECISION_TYPE_OPTIONS, defaultValue: 'transfer', required: true },
    { field: 'outcome', label: 'Rezultat final', type: 'select', options: MOBILITY_FINAL_DECISION_OUTCOME_OPTIONS, defaultValue: 'admis', required: true },
    { field: 'approved_on', label: 'Data aprobarii', type: 'date', required: true },
    { field: 'effective_from', label: 'Data aplicarii', type: 'date', required: true },
    { field: 'panel_name', label: 'Comisie / organism', type: 'text', wide: true, required: true },
    { field: 'destination_unit', label: 'Unitate destinatie', type: 'text' },
    { field: 'legal_basis', label: 'Baza legala', type: 'textarea', wide: true },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista decizie finala pentru cazul de mobilitate selectat.',
};

const MOBILITY_RESULT_ISSUES_CHILD: EducationDetailChildResourceConfig = {
  key: 'mobility_result_issues',
  label: 'Emitere rezultat',
  icon: 'pi pi-send',
  description: 'Comunicarea oficiala a rezultatului de mobilitate, cu registratura si status de livrare.',
  listEndpoint: (parentRow) => `/api/education/mobility/records/${parentRow['id']}/result-issues`,
  detailEndpoint: (parentRow, childRow) => `/api/education/mobility/records/${parentRow['id']}/result-issues/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/mobility/records/${parentRow['id']}/result-issues`,
  pdfEndpoint: (parentRow, childRow) => `/api/education/mobility/records/${parentRow['id']}/result-issues/${childRow['id']}/pdf`,
  pdfFilename: (_parentRow, childRow) => `comunicare-mobilitate-${childRow['issue_code'] ?? childRow['id']}.pdf`,
  pdfActionLabel: 'Comunicare PDF',
  readPermission: 'education.mobility.read',
  managePermission: 'education.mobility.manage',
  columns: [
    { field: 'issue_code', header: 'Cod', sortable: true, filter: 'text', width: '12rem' },
    { field: 'document_type', header: 'Document', type: 'tag', sortable: true, filter: 'select', options: RESULT_DOCUMENT_TYPE_OPTIONS, width: '12rem' },
    { field: 'recipient_name', header: 'Destinatar', sortable: true, filter: 'text', width: '18rem' },
    { field: 'recipient_role', header: 'Rol', sortable: true, filter: 'text', width: '14rem' },
    { field: 'delivery_channel', header: 'Canal', type: 'tag', sortable: true, filter: 'select', options: RESULT_DELIVERY_CHANNEL_OPTIONS, width: '10rem' },
    { field: 'delivery_status', header: 'Livrare', type: 'tag', sortable: true, filter: 'select', options: RESULT_DELIVERY_STATUS_OPTIONS, width: '10rem' },
    { field: 'issued_on', header: 'Emis la', sortable: true, width: '9rem' },
    { field: 'delivered_on', header: 'Predat la', sortable: true, width: '9rem' },
  ],
  createFields: [
    { field: 'document_type', label: 'Tip document', type: 'select', options: RESULT_DOCUMENT_TYPE_OPTIONS, defaultValue: 'decizie', required: true },
    { field: 'recipient_name', label: 'Destinatar', type: 'text', wide: true, required: true },
    { field: 'recipient_role', label: 'Rol destinatar', type: 'text' },
    { field: 'delivery_channel', label: 'Canal livrare', type: 'select', options: RESULT_DELIVERY_CHANNEL_OPTIONS, defaultValue: 'registratura', required: true },
    { field: 'delivery_status', label: 'Status livrare', type: 'select', options: RESULT_DELIVERY_STATUS_OPTIONS, defaultValue: 'pregatit', required: true },
    { field: 'issued_on', label: 'Data emiterii', type: 'date', required: true },
    { field: 'delivered_on', label: 'Data predarii', type: 'date' },
    { field: 'registry_reference', label: 'Referinta registratura', type: 'text' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista emiteri de rezultat pentru cazul de mobilitate selectat.',
};

const MERIT_DOCUMENTS_CHILD: EducationDetailChildResourceConfig = {
  key: 'merit_documents',
  label: 'Dosar gradatie',
  icon: 'pi pi-folder',
  description: 'Cerere, declaratii, autoevaluare si anexe din dosarul pentru gradatia de merit.',
  listEndpoint: (parentRow) => `/api/education/gradatii/records/${parentRow['id']}/documents`,
  detailEndpoint: (parentRow, childRow) => `/api/education/gradatii/records/${parentRow['id']}/documents/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/gradatii/records/${parentRow['id']}/documents`,
  readPermission: 'education.gradatii.read',
  managePermission: 'education.gradatii.manage',
  columns: [
    { field: 'document_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'document_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: MERIT_DOCUMENT_TYPE_OPTIONS, width: '12rem' },
    { field: 'document_title', header: 'Document', sortable: true, filter: 'text', width: '22rem' },
    { field: 'validation_status', header: 'Validare', type: 'tag', sortable: true, filter: 'select', options: DOSSIER_VALIDATION_OPTIONS, width: '10rem' },
    { field: 'submitted_by', header: 'Depus de', sortable: true, filter: 'text', width: '14rem' },
    { field: 'registered_on', header: 'Inregistrat', sortable: true, width: '9rem' },
    { field: 'mandatory', header: 'Obligatoriu', type: 'boolean', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'document_type', label: 'Tip document', type: 'select', options: MERIT_DOCUMENT_TYPE_OPTIONS, defaultValue: 'cerere', required: true },
    { field: 'document_title', label: 'Titlu document', type: 'text', wide: true, required: true },
    { field: 'registered_on', label: 'Data inregistrarii', type: 'date', required: true },
    { field: 'submitted_by', label: 'Depus de', type: 'text' },
    { field: 'validation_status', label: 'Status validare', type: 'select', options: DOSSIER_VALIDATION_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'mandatory', label: 'Document obligatoriu', type: 'boolean', defaultValue: true },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista documente in dosarul de gradatie selectat.',
};

const MERIT_SCORES_CHILD: EducationDetailChildResourceConfig = {
  key: 'merit_scores',
  label: 'Evaluare comisie',
  icon: 'pi pi-chart-bar',
  description: 'Punctaj pe criterii si etape de evaluare pentru gradatia de merit.',
  listEndpoint: (parentRow) => `/api/education/gradatii/records/${parentRow['id']}/scores`,
  detailEndpoint: (parentRow, childRow) => `/api/education/gradatii/records/${parentRow['id']}/scores/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/gradatii/records/${parentRow['id']}/scores`,
  readPermission: 'education.gradatii.read',
  managePermission: 'education.gradatii.manage',
  columns: [
    { field: 'criterion_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
    { field: 'criterion_category', header: 'Categorie', type: 'tag', sortable: true, filter: 'select', options: MERIT_CRITERION_CATEGORY_OPTIONS, width: '12rem' },
    { field: 'panel_stage', header: 'Etapa', type: 'tag', sortable: true, filter: 'select', options: MERIT_PANEL_STAGE_OPTIONS, width: '12rem' },
    { field: 'criterion_label', header: 'Criteriu', sortable: true, filter: 'text', width: '22rem' },
    { field: 'max_score', header: 'Maxim', type: 'number', sortable: true, width: '7rem' },
    { field: 'awarded_score', header: 'Acordat', type: 'number', sortable: true, width: '7rem' },
    { field: 'contested', header: 'Contestat', type: 'boolean', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'criterion_code', label: 'Cod criteriu', type: 'text', required: true },
    { field: 'criterion_label', label: 'Denumire criteriu', type: 'text', wide: true, required: true },
    { field: 'criterion_category', label: 'Categorie', type: 'select', options: MERIT_CRITERION_CATEGORY_OPTIONS, defaultValue: 'performanta', required: true },
    { field: 'panel_stage', label: 'Etapa evaluare', type: 'select', options: MERIT_PANEL_STAGE_OPTIONS, defaultValue: 'evaluare_comisie', required: true },
    { field: 'max_score', label: 'Punctaj maxim', type: 'number', defaultValue: 10, required: true },
    { field: 'awarded_score', label: 'Punctaj acordat', type: 'number', defaultValue: 0, required: true },
    { field: 'reviewer_name', label: 'Evaluator', type: 'text' },
    { field: 'evidence_reference', label: 'Referinta dovada', type: 'text' },
    { field: 'contested', label: 'Contestat', type: 'boolean', defaultValue: false },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista punctaje pe criterii pentru gradatia selectata.',
};

const MERIT_APPEALS_CHILD: EducationDetailChildResourceConfig = {
  key: 'merit_appeals',
  label: 'Contestatii',
  icon: 'pi pi-megaphone',
  description: 'Contestatii privind evaluarea pentru gradatia de merit si decizia de solutionare.',
  listEndpoint: (parentRow) => `/api/education/gradatii/records/${parentRow['id']}/appeals`,
  detailEndpoint: (parentRow, childRow) => `/api/education/gradatii/records/${parentRow['id']}/appeals/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/gradatii/records/${parentRow['id']}/appeals`,
  pdfEndpoint: (parentRow, childRow) => `/api/education/gradatii/records/${parentRow['id']}/appeals/${childRow['id']}/pdf`,
  pdfFilename: (_parentRow, childRow) => `contestatie-gradatie-${childRow['appeal_code'] ?? childRow['id']}.pdf`,
  pdfActionLabel: 'Contestatie PDF',
  readPermission: 'education.gradatii.read',
  managePermission: 'education.gradatii.manage',
  columns: [
    { field: 'appeal_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'submitted_by', header: 'Depusa de', sortable: true, filter: 'text', width: '16rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: APPEAL_STATUS_OPTIONS, width: '10rem' },
    { field: 'submitted_on', header: 'Depusa la', sortable: true, width: '9rem' },
    { field: 'resolved_on', header: 'Solutionata', sortable: true, width: '10rem' },
  ],
  createFields: [
    { field: 'submitted_by', label: 'Depusa de', type: 'text', wide: true, required: true },
    { field: 'submitted_on', label: 'Data depunerii', type: 'date', required: true },
    { field: 'status', label: 'Status contestatie', type: 'select', options: APPEAL_STATUS_OPTIONS, defaultValue: 'submitted', required: true },
    { field: 'grounds', label: 'Motive contestatie', type: 'textarea', wide: true, required: true },
    { field: 'resolved_on', label: 'Data solutionarii', type: 'date' },
    { field: 'decision_summary', label: 'Rezumat decizie', type: 'textarea', wide: true },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista contestatii pentru gradatia selectata.',
};

const MERIT_FINAL_DECISIONS_CHILD: EducationDetailChildResourceConfig = {
  key: 'merit_final_decisions',
  label: 'Decizie finala',
  icon: 'pi pi-verified',
  description: 'Validarea finala a gradatiei de merit, cu etapa decizionala, finantare si baza legala.',
  listEndpoint: (parentRow) => `/api/education/gradatii/records/${parentRow['id']}/final-decisions`,
  detailEndpoint: (parentRow, childRow) => `/api/education/gradatii/records/${parentRow['id']}/final-decisions/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/gradatii/records/${parentRow['id']}/final-decisions`,
  pdfEndpoint: (parentRow, childRow) => `/api/education/gradatii/records/${parentRow['id']}/final-decisions/${childRow['id']}/pdf`,
  pdfFilename: (_parentRow, childRow) => `decizie-gradatie-${childRow['decision_code'] ?? childRow['id']}.pdf`,
  pdfActionLabel: 'Decizie PDF',
  readPermission: 'education.gradatii.read',
  managePermission: 'education.gradatii.manage',
  columns: [
    { field: 'decision_code', header: 'Cod', sortable: true, filter: 'text', width: '12rem' },
    { field: 'decision_stage', header: 'Etapa', type: 'tag', sortable: true, filter: 'select', options: MERIT_FINAL_DECISION_STAGE_OPTIONS, width: '14rem' },
    { field: 'outcome', header: 'Rezultat', type: 'tag', sortable: true, filter: 'select', options: MERIT_FINAL_DECISION_OUTCOME_OPTIONS, width: '10rem' },
    { field: 'panel_name', header: 'Comisie', sortable: true, filter: 'text', width: '18rem' },
    { field: 'funded', header: 'Finantat', type: 'boolean', sortable: true, width: '8rem' },
    { field: 'approved_on', header: 'Aprobata la', sortable: true, width: '9rem' },
    { field: 'effective_from', header: 'Aplicare de la', sortable: true, width: '10rem' },
  ],
  createFields: [
    { field: 'decision_stage', label: 'Etapa deciziei', type: 'select', options: MERIT_FINAL_DECISION_STAGE_OPTIONS, defaultValue: 'validare_finala', required: true },
    { field: 'outcome', label: 'Rezultat final', type: 'select', options: MERIT_FINAL_DECISION_OUTCOME_OPTIONS, defaultValue: 'admis', required: true },
    { field: 'approved_on', label: 'Data aprobarii', type: 'date', required: true },
    { field: 'effective_from', label: 'Data aplicarii', type: 'date', required: true },
    { field: 'panel_name', label: 'Comisie / organism', type: 'text', wide: true, required: true },
    { field: 'funded', label: 'Include finantare', type: 'boolean', defaultValue: false },
    { field: 'legal_basis', label: 'Baza legala', type: 'textarea', wide: true },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista decizie finala pentru gradatia selectata.',
};

const MERIT_RESULT_ISSUES_CHILD: EducationDetailChildResourceConfig = {
  key: 'merit_result_issues',
  label: 'Emitere rezultat',
  icon: 'pi pi-send',
  description: 'Comunicarea rezultatului final al gradatiei de merit, cu canal si confirmare de livrare.',
  listEndpoint: (parentRow) => `/api/education/gradatii/records/${parentRow['id']}/result-issues`,
  detailEndpoint: (parentRow, childRow) => `/api/education/gradatii/records/${parentRow['id']}/result-issues/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/gradatii/records/${parentRow['id']}/result-issues`,
  pdfEndpoint: (parentRow, childRow) => `/api/education/gradatii/records/${parentRow['id']}/result-issues/${childRow['id']}/pdf`,
  pdfFilename: (_parentRow, childRow) => `comunicare-gradatie-${childRow['issue_code'] ?? childRow['id']}.pdf`,
  pdfActionLabel: 'Comunicare PDF',
  readPermission: 'education.gradatii.read',
  managePermission: 'education.gradatii.manage',
  columns: [
    { field: 'issue_code', header: 'Cod', sortable: true, filter: 'text', width: '12rem' },
    { field: 'document_type', header: 'Document', type: 'tag', sortable: true, filter: 'select', options: MERIT_RESULT_DOCUMENT_TYPE_OPTIONS, width: '12rem' },
    { field: 'recipient_name', header: 'Destinatar', sortable: true, filter: 'text', width: '18rem' },
    { field: 'recipient_role', header: 'Rol', sortable: true, filter: 'text', width: '14rem' },
    { field: 'delivery_channel', header: 'Canal', type: 'tag', sortable: true, filter: 'select', options: RESULT_DELIVERY_CHANNEL_OPTIONS, width: '10rem' },
    { field: 'delivery_status', header: 'Livrare', type: 'tag', sortable: true, filter: 'select', options: RESULT_DELIVERY_STATUS_OPTIONS, width: '10rem' },
    { field: 'issued_on', header: 'Emis la', sortable: true, width: '9rem' },
    { field: 'delivered_on', header: 'Predat la', sortable: true, width: '9rem' },
  ],
  createFields: [
    { field: 'document_type', label: 'Tip document', type: 'select', options: MERIT_RESULT_DOCUMENT_TYPE_OPTIONS, defaultValue: 'comunicare', required: true },
    { field: 'recipient_name', label: 'Destinatar', type: 'text', wide: true, required: true },
    { field: 'recipient_role', label: 'Rol destinatar', type: 'text' },
    { field: 'delivery_channel', label: 'Canal livrare', type: 'select', options: RESULT_DELIVERY_CHANNEL_OPTIONS, defaultValue: 'email', required: true },
    { field: 'delivery_status', label: 'Status livrare', type: 'select', options: RESULT_DELIVERY_STATUS_OPTIONS, defaultValue: 'pregatit', required: true },
    { field: 'issued_on', label: 'Data emiterii', type: 'date', required: true },
    { field: 'delivered_on', label: 'Data predarii', type: 'date' },
    { field: 'registry_reference', label: 'Referinta registratura', type: 'text' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista emiteri de rezultat pentru gradatia selectata.',
};

const PORTFOLIO_DOCUMENTS_CHILD: EducationDetailChildResourceConfig = {
  key: 'portfolio_documents',
  label: 'Documente portofoliu',
  icon: 'pi pi-folder',
  description: 'Documente organizate pe sectiuni si componente, cu cronologie si separare portofoliu vs dosar personal.',
  listEndpoint: (parentRow) => `/api/education/portfolios/records/${parentRow['id']}/documents`,
  detailEndpoint: (parentRow, childRow) => `/api/education/portfolios/records/${parentRow['id']}/documents/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/portfolios/records/${parentRow['id']}/documents`,
  readPermission: 'education.portfolios.read',
  managePermission: 'education.portfolios.manage',
  columns: [
    { field: 'section_code', header: 'Sectiune', sortable: true, filter: 'text', width: '10rem' },
    { field: 'component_code', header: 'Componenta', sortable: true, filter: 'text', width: '12rem' },
    { field: 'document_title', header: 'Document', sortable: true, filter: 'text', width: '20rem' },
    { field: 'source_scope', header: 'Sursa', type: 'tag', sortable: true, filter: 'select', options: PORTFOLIO_DOCUMENT_SOURCE_OPTIONS, width: '12rem' },
    { field: 'authenticity_status', header: 'Autenticitate', type: 'tag', sortable: true, filter: 'select', options: PORTFOLIO_DOCUMENT_AUTHENTICITY_OPTIONS, width: '11rem' },
    { field: 'issued_on', header: 'Emis la', sortable: true, filter: 'text', width: '9rem' },
    { field: 'chronological_index', header: 'Ordine', type: 'number', sortable: true, width: '7rem' },
    { field: 'sensitive_data', header: 'Sensibil', type: 'boolean', sortable: true, width: '7rem' },
  ],
  createFields: [
    { field: 'section_code', label: 'Cod sectiune', type: 'text', required: true },
    { field: 'component_code', label: 'Cod componenta', type: 'text', required: true },
    { field: 'document_title', label: 'Titlu document', type: 'text', wide: true, required: true },
    { field: 'source_scope', label: 'Apartine', type: 'select', options: PORTFOLIO_DOCUMENT_SOURCE_OPTIONS, defaultValue: 'portofoliu', required: true },
    { field: 'evidence_type', label: 'Tip dovada', type: 'select', options: PORTFOLIO_DOCUMENT_EVIDENCE_OPTIONS, defaultValue: 'altele', required: true },
    { field: 'issued_on', label: 'Data emiterii', type: 'date', required: true },
    { field: 'added_on', label: 'Data adaugarii', type: 'date', required: true },
    { field: 'chronological_index', label: 'Ordine cronologica', type: 'number', defaultValue: 0 },
    { field: 'sensitive_data', label: 'Contine date sensibile', type: 'boolean', defaultValue: false },
    { field: 'authenticity_status', label: 'Status autenticitate', type: 'select', options: PORTFOLIO_DOCUMENT_AUTHENTICITY_OPTIONS, defaultValue: 'declarat', required: true },
    { field: 'file_reference', label: 'Referinta fisier / registratura', type: 'text' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista documente pentru portofoliul selectat.',
};

const PORTFOLIO_CHECKLIST_CHILD: EducationDetailChildResourceConfig = {
  key: 'portfolio_checklist',
  label: 'Checklist conformitate',
  icon: 'pi pi-list-check',
  description: 'Opis procedural, documente obligatorii, stare de completare si verificare pe sectiuni.',
  listEndpoint: (parentRow) => `/api/education/portfolios/records/${parentRow['id']}/checklist`,
  detailEndpoint: (parentRow, childRow) => `/api/education/portfolios/records/${parentRow['id']}/checklist/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/portfolios/records/${parentRow['id']}/checklist`,
  readPermission: 'education.portfolios.read',
  managePermission: 'education.portfolios.manage',
  columns: [
    { field: 'requirement_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
    { field: 'requirement_label', header: 'Cerinta', sortable: true, filter: 'text', width: '20rem' },
    { field: 'section_code', header: 'Sectiune', sortable: true, filter: 'text', width: '10rem' },
    { field: 'source_scope', header: 'Sursa', type: 'tag', sortable: true, filter: 'select', options: PORTFOLIO_DOCUMENT_SOURCE_OPTIONS, width: '11rem' },
    { field: 'mandatory', header: 'Obligatoriu', type: 'boolean', sortable: true, width: '8rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: PORTFOLIO_CHECKLIST_STATUS_OPTIONS, width: '10rem' },
    { field: 'document_count', header: 'Doc.', type: 'number', sortable: true, width: '6rem' },
    { field: 'last_checked_on', header: 'Verificat la', sortable: true, width: '10rem' },
  ],
  createFields: [
    { field: 'requirement_code', label: 'Cod cerinta', type: 'text', required: true },
    { field: 'requirement_label', label: 'Denumire cerinta', type: 'text', wide: true, required: true },
    { field: 'section_code', label: 'Sectiune', type: 'text', required: true },
    { field: 'source_scope', label: 'Sursa documentelor', type: 'select', options: PORTFOLIO_DOCUMENT_SOURCE_OPTIONS, defaultValue: 'portofoliu', required: true },
    { field: 'mandatory', label: 'Obligatoriu', type: 'boolean', defaultValue: true },
    { field: 'status', label: 'Status conformitate', type: 'select', options: PORTFOLIO_CHECKLIST_STATUS_OPTIONS, defaultValue: 'partial', required: true },
    { field: 'document_count', label: 'Numar documente', type: 'number', defaultValue: 0 },
    { field: 'last_checked_on', label: 'Ultima verificare', type: 'date', required: true },
    { field: 'checked_by', label: 'Verificat de', type: 'text' },
    { field: 'notes', label: 'Observatii', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista checklist de conformitate pentru portofoliul selectat.',
};

const PORTFOLIO_TRANSFERS_CHILD: EducationDetailChildResourceConfig = {
  key: 'portfolio_transfers',
  label: 'Predare si transfer',
  icon: 'pi pi-send',
  description: 'Circuitul de predare-primire al portofoliului intre institutii si responsabili.',
  listEndpoint: (parentRow) => `/api/education/portfolios/records/${parentRow['id']}/transfers`,
  detailEndpoint: (parentRow, childRow) => `/api/education/portfolios/records/${parentRow['id']}/transfers/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/portfolios/records/${parentRow['id']}/transfers`,
  readPermission: 'education.portfolios.read',
  managePermission: 'education.portfolios.manage',
  columns: [
    { field: 'transfer_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'transfer_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: PORTFOLIO_TRANSFER_TYPE_OPTIONS, width: '10rem' },
    { field: 'source_institution', header: 'Institutie sursa', sortable: true, filter: 'text', width: '18rem' },
    { field: 'destination_institution', header: 'Institutie destinatie', sortable: true, filter: 'text', width: '18rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: PORTFOLIO_TRANSFER_STATUS_OPTIONS, width: '10rem' },
    { field: 'handover_on', header: 'Predat la', sortable: true, width: '9rem' },
    { field: 'received_on', header: 'Receptionat la', sortable: true, width: '10rem' },
  ],
  createFields: [
    { field: 'transfer_type', label: 'Tip transfer', type: 'select', options: PORTFOLIO_TRANSFER_TYPE_OPTIONS, defaultValue: 'predare', required: true },
    { field: 'source_institution', label: 'Institutie sursa', type: 'text', wide: true, required: true },
    { field: 'destination_institution', label: 'Institutie destinatie', type: 'text', wide: true, required: true },
    { field: 'status', label: 'Status transfer', type: 'select', options: PORTFOLIO_TRANSFER_STATUS_OPTIONS, defaultValue: 'pregatit', required: true },
    { field: 'handover_on', label: 'Data predarii', type: 'date', required: true },
    { field: 'received_on', label: 'Data receptionarii', type: 'date' },
    { field: 'handover_by', label: 'Predat de', type: 'text' },
    { field: 'received_by', label: 'Receptionat de', type: 'text' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista evenimente de predare-primire pentru portofoliul selectat.',
};

const PORTFOLIO_REVIEWS_CHILD: EducationDetailChildResourceConfig = {
  key: 'portfolio_reviews',
  label: 'Verificari si validari',
  icon: 'pi pi-check-circle',
  description: 'Etape de verificare, completari solicitate si scor de conformitate pentru portofoliu.',
  listEndpoint: (parentRow) => `/api/education/portfolios/records/${parentRow['id']}/reviews`,
  detailEndpoint: (parentRow, childRow) => `/api/education/portfolios/records/${parentRow['id']}/reviews/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/portfolios/records/${parentRow['id']}/reviews`,
  readPermission: 'education.portfolios.read',
  managePermission: 'education.portfolios.manage',
  columns: [
    { field: 'review_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'review_stage', header: 'Etapa', type: 'tag', sortable: true, filter: 'select', options: PORTFOLIO_REVIEW_STAGE_OPTIONS, width: '14rem' },
    { field: 'outcome', header: 'Rezultat', type: 'tag', sortable: true, filter: 'select', options: PORTFOLIO_REVIEW_OUTCOME_OPTIONS, width: '10rem' },
    { field: 'reviewer_name', header: 'Evaluator', sortable: true, filter: 'text', width: '16rem' },
    { field: 'reviewed_on', header: 'Data', sortable: true, width: '9rem' },
    { field: 'missing_documents', header: 'Lipsuri', type: 'number', sortable: true, width: '7rem' },
    { field: 'compliance_score', header: 'Scor', type: 'number', sortable: true, width: '7rem' },
  ],
  createFields: [
    { field: 'review_stage', label: 'Etapa verificarii', type: 'select', options: PORTFOLIO_REVIEW_STAGE_OPTIONS, defaultValue: 'verificare_secretariat', required: true },
    { field: 'outcome', label: 'Rezultat', type: 'select', options: PORTFOLIO_REVIEW_OUTCOME_OPTIONS, defaultValue: 'completari', required: true },
    { field: 'reviewer_name', label: 'Evaluator', type: 'text', required: true },
    { field: 'reviewed_on', label: 'Data verificarii', type: 'date', required: true },
    { field: 'missing_documents', label: 'Documente lipsa', type: 'number', defaultValue: 0 },
    { field: 'compliance_score', label: 'Scor conformitate', type: 'number', defaultValue: 0 },
    { field: 'notes', label: 'Concluzii', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista verificari inregistrate pentru portofoliul selectat.',
};

const PORTFOLIO_VALORIFICATIONS_CHILD: EducationDetailChildResourceConfig = {
  key: 'portfolio_valorifications',
  label: 'Fluxuri de valorificare',
  icon: 'pi pi-sitemap',
  description: 'Situatiile procedurale in care portofoliul este folosit, transmis, evaluat sau prezentat institutional.',
  listEndpoint: (parentRow) => `/api/education/portfolios/records/${parentRow['id']}/valorifications`,
  detailEndpoint: (parentRow, childRow) => `/api/education/portfolios/records/${parentRow['id']}/valorifications/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/portfolios/records/${parentRow['id']}/valorifications`,
  readPermission: 'education.portfolios.read',
  managePermission: 'education.portfolios.manage',
  columns: [
    { field: 'valorification_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'scope', header: 'Scop', type: 'tag', sortable: true, filter: 'select', options: PORTFOLIO_VALORIFICATION_SCOPE_OPTIONS, width: '16rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: PORTFOLIO_VALORIFICATION_STATUS_OPTIONS, width: '10rem' },
    { field: 'requested_by', header: 'Initiat de', sortable: true, filter: 'text', width: '14rem' },
    { field: 'target_institution', header: 'Tinta institutionala', sortable: true, filter: 'text', width: '18rem' },
    { field: 'target_reference', header: 'Referinta', sortable: true, filter: 'text', width: '12rem' },
    { field: 'started_on', header: 'Pornit la', sortable: true, width: '9rem' },
    { field: 'completed_on', header: 'Finalizat la', sortable: true, width: '10rem' },
  ],
  createFields: [
    { field: 'scope', label: 'Scop procedural', type: 'select', options: PORTFOLIO_VALORIFICATION_SCOPE_OPTIONS, defaultValue: 'evaluare_profesionala', required: true },
    { field: 'status', label: 'Status', type: 'select', options: PORTFOLIO_VALORIFICATION_STATUS_OPTIONS, defaultValue: 'planificat', required: true },
    { field: 'requested_by', label: 'Initiat de', type: 'text' },
    { field: 'target_institution', label: 'Institutie / comisie tinta', type: 'text', wide: true },
    { field: 'target_reference', label: 'Referinta procedurala', type: 'text' },
    { field: 'started_on', label: 'Data pornirii', type: 'date', required: true },
    { field: 'completed_on', label: 'Data finalizarii', type: 'date' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista fluxuri de valorificare pentru portofoliul selectat.',
};

const PORTFOLIO_OPIS_CHILD: EducationDetailChildResourceConfig = {
  key: 'portfolio_opis',
  label: 'Opis si index cronologic',
  icon: 'pi pi-sort-numeric-down',
  description: 'Index real al documentelor din portofoliu, cu ordine cronologica si referinte trasabile.',
  listEndpoint: (parentRow) => `/api/education/portfolios/records/${parentRow['id']}/opis`,
  detailEndpoint: (parentRow, childRow) => `/api/education/portfolios/records/${parentRow['id']}/opis/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/portfolios/records/${parentRow['id']}/opis`,
  readPermission: 'education.portfolios.read',
  managePermission: 'education.portfolios.manage',
  columns: [
    { field: 'chronological_index', header: 'Ordine', type: 'number', sortable: true, width: '7rem' },
    { field: 'section_code', header: 'Sectiune', sortable: true, filter: 'text', width: '10rem' },
    { field: 'component_code', header: 'Componenta', sortable: true, filter: 'text', width: '12rem' },
    { field: 'entry_title', header: 'Titlu indexat', sortable: true, filter: 'text', width: '20rem' },
    { field: 'source_scope', header: 'Sursa', type: 'tag', sortable: true, filter: 'select', options: PORTFOLIO_DOCUMENT_SOURCE_OPTIONS, width: '11rem' },
    { field: 'document_reference', header: 'Referinta', sortable: true, filter: 'text', width: '12rem' },
    { field: 'included_in_transfer', header: 'La transfer', type: 'boolean', sortable: true, width: '8rem' },
    { field: 'checked_on', header: 'Verificat', sortable: true, width: '9rem' },
  ],
  createFields: [
    { field: 'section_code', label: 'Cod sectiune', type: 'text', required: true },
    { field: 'component_code', label: 'Cod componenta', type: 'text', required: true },
    { field: 'entry_title', label: 'Titlu indexat', type: 'text', wide: true, required: true },
    { field: 'source_scope', label: 'Sursa', type: 'select', options: PORTFOLIO_DOCUMENT_SOURCE_OPTIONS, defaultValue: 'portofoliu', required: true },
    { field: 'chronological_index', label: 'Ordine cronologica', type: 'number', defaultValue: 0, required: true },
    { field: 'document_reference', label: 'Referinta document', type: 'text', required: true },
    { field: 'included_in_transfer', label: 'Inclus la transfer', type: 'boolean', defaultValue: false },
    { field: 'checked_on', label: 'Verificat la', type: 'date', required: true },
    { field: 'checked_by', label: 'Verificat de', type: 'text' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista opis structurat pentru portofoliul selectat.',
};

const PORTFOLIO_CUSTODY_CHILD: EducationDetailChildResourceConfig = {
  key: 'portfolio_custody',
  label: 'Custodie si acces',
  icon: 'pi pi-lock',
  description: 'Jurnal de preluare, consultare, transfer si arhivare pentru portofoliu si date sensibile.',
  listEndpoint: (parentRow) => `/api/education/portfolios/records/${parentRow['id']}/custody`,
  detailEndpoint: (parentRow, childRow) => `/api/education/portfolios/records/${parentRow['id']}/custody/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/portfolios/records/${parentRow['id']}/custody`,
  readPermission: 'education.portfolios.read',
  managePermission: 'education.portfolios.manage',
  columns: [
    { field: 'event_type', header: 'Eveniment', type: 'tag', sortable: true, filter: 'select', options: PORTFOLIO_CUSTODY_EVENT_OPTIONS, width: '11rem' },
    { field: 'holder_name', header: 'Detinator', sortable: true, filter: 'text', width: '16rem' },
    { field: 'holder_role', header: 'Rol', sortable: true, filter: 'text', width: '14rem' },
    { field: 'location_label', header: 'Locatie', sortable: true, filter: 'text', width: '14rem' },
    { field: 'started_on', header: 'De la', sortable: true, width: '9rem' },
    { field: 'ended_on', header: 'Pana la', sortable: true, width: '9rem' },
    { field: 'access_mode', header: 'Acces', type: 'tag', sortable: true, filter: 'select', options: PORTFOLIO_CUSTODY_ACCESS_MODE_OPTIONS, width: '10rem' },
    { field: 'sensitive_data_access', header: 'Date sensibile', type: 'boolean', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'event_type', label: 'Tip eveniment', type: 'select', options: PORTFOLIO_CUSTODY_EVENT_OPTIONS, defaultValue: 'preluare', required: true },
    { field: 'holder_name', label: 'Detinator / accesant', type: 'text', wide: true, required: true },
    { field: 'holder_role', label: 'Rol', type: 'text', required: true },
    { field: 'location_label', label: 'Locatie / compartiment', type: 'text', required: true },
    { field: 'access_reason', label: 'Motiv acces', type: 'textarea', wide: true, required: true },
    { field: 'started_on', label: 'Data inceput', type: 'date', required: true },
    { field: 'ended_on', label: 'Data sfarsit', type: 'date' },
    { field: 'access_mode', label: 'Mod acces', type: 'select', options: PORTFOLIO_CUSTODY_ACCESS_MODE_OPTIONS, defaultValue: 'fizic', required: true },
    { field: 'sensitive_data_access', label: 'Acces la date sensibile', type: 'boolean', defaultValue: false },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista jurnal de custodie pentru portofoliul selectat.',
};

const DASHBOARD_CARDS: EducationDashboardCardConfig[] = [
  {
    key: 'meetings',
    label: 'Sedinte guvernanta',
    icon: 'pi pi-calendar',
    endpoint: '/api/education/dashboard',
    statKey: 'total_meetings',
    permission: 'education.governance.read',
  },
  {
    key: 'decisions',
    label: 'Decizii CA/CP',
    icon: 'pi pi-verified',
    endpoint: '/api/education/decisions/dashboard',
    statKey: 'total_decisions',
    permission: 'education.decisions.read',
  },
  {
    key: 'managerial',
    label: 'Dosare manageriale',
    icon: 'pi pi-briefcase',
    endpoint: '/api/education/managerial/dashboard',
    statKey: 'total_dossiers',
    permission: 'education.managerial.read',
  },
  {
    key: 'personnel',
    label: 'Personal activ',
    icon: 'pi pi-id-card',
    endpoint: '/api/education/personnel/dashboard',
    statKey: 'active_records',
    permission: 'education.personnel.read',
  },
  {
    key: 'portfolios',
    label: 'Portofolii validate',
    icon: 'pi pi-folder-open',
    endpoint: '/api/education/portfolios/dashboard',
    statKey: 'validated_portfolios',
    permission: 'education.portfolios.read',
  },
  {
    key: 'mobility',
    label: 'Rezultate mobilitate comunicate',
    icon: 'pi pi-send',
    endpoint: '/api/education/mobility/dashboard',
    statKey: 'communicated_results',
    permission: 'education.mobility.read',
  },
  {
    key: 'merit',
    label: 'Gradații finanțate',
    icon: 'pi pi-star',
    endpoint: '/api/education/gradatii/dashboard',
    statKey: 'funded_records',
    permission: 'education.gradatii.read',
  },
  {
    key: 'evaluations',
    label: 'Evaluari comunicate',
    icon: 'pi pi-megaphone',
    endpoint: '/api/education/evaluations/dashboard',
    statKey: 'communicated_results',
    permission: 'education.evaluations.read',
  },
];

const REQUIREMENTS_RESOURCE: EducationResourceConfig = {
  key: 'requirements',
  label: 'Cerinte legale',
  icon: 'pi pi-bookmark',
  endpoint: '/api/education/requirements',
  createEndpoint: '',
  allowCreate: false,
  readPermission: 'education.read',
  description: 'Catalog trasabil din legislatia educationala, ROFUIP si metodologiile tematice.',
  columns: [
    { field: 'priority', header: 'Prioritate', type: 'number', sortable: true, width: '7rem' },
    { field: 'domain', header: 'Domeniu', sortable: true, filter: 'text', width: '10rem' },
    { field: 'title_ro', header: 'Cerinta', sortable: true, filter: 'text', width: '28rem' },
    { field: 'source_ref', header: 'Sursa', sortable: true, filter: 'text', width: '18rem' },
    { field: 'requirement_type', header: 'Tip', type: 'tag', sortable: true, filter: 'text', width: '12rem' },
    {
      field: 'implementation_status',
      header: 'Implementare',
      type: 'tag',
      sortable: true,
      filter: 'select',
      options: [
        { label: 'Implementat', value: 'implemented' },
        { label: 'Partial', value: 'partial' },
        { label: 'Planificat', value: 'planned' },
      ],
      width: '10rem',
    },
  ],
  createFields: [],
  emptyText: 'Catalogul de cerinte nu contine inca inregistrari.',
};

const MEETINGS_RESOURCE: EducationResourceConfig = {
  key: 'meetings',
  label: 'Sedinte CA/CP/CEAC',
  icon: 'pi pi-calendar',
  endpoint: '/api/education/governance/meetings',
  createEndpoint: '/api/education/governance/meetings',
  createWizardRoute: '/education/governance/ca-wizard',
  detailSummaryKind: 'governance-finalization',
  detailSummaryEndpoint: (row) => `/api/education/governance/meetings/${String(row['id'] ?? '')}/finalization-summary`,
  readPermission: 'education.governance.read',
  managePermission: 'education.governance.manage',
  description: 'Convocare, prezenta, cvorum, agenda, minute, anexe, vot si semnaturi.',
  columns: [
    { field: 'school_year', header: 'An scolar', sortable: true, filter: 'text', width: '9rem' },
    { field: 'organism', header: 'Organism', type: 'tag', sortable: true, filter: 'select', options: ORGANISM_OPTIONS, width: '10rem' },
    { field: 'title', header: 'Titlu', sortable: true, filter: 'text', width: '20rem' },
    { field: 'meeting_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: MEETING_TYPE_OPTIONS, width: '10rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: MEETING_STATUS_OPTIONS, width: '10rem' },
    { field: 'meeting_date', header: 'Data', sortable: true, width: '9rem' },
    { field: 'participants_count', header: 'Participanti', type: 'number', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'school_year', label: 'An scolar', type: 'text', defaultValue: '2025-2026', required: true },
    { field: 'organism', label: 'Organism', type: 'select', options: ORGANISM_OPTIONS, defaultValue: 'ca', required: true },
    { field: 'title', label: 'Titlu', type: 'text', required: true, wide: true },
    { field: 'meeting_type', label: 'Tip sedinta', type: 'select', options: MEETING_TYPE_OPTIONS, defaultValue: 'ordinary', required: true },
    { field: 'status', label: 'Status', type: 'select', options: MEETING_STATUS_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'quorum_required', label: 'Cvorum necesar', type: 'number', defaultValue: 1 },
    { field: 'participants_count', label: 'Participanti', type: 'number', defaultValue: 0 },
    { field: 'meeting_date', label: 'Data sedintei', type: 'date' },
    { field: 'location', label: 'Locatie', type: 'text' },
    { field: 'chairperson', label: 'Presedinte', type: 'text' },
    { field: 'secretary_name', label: 'Secretar', type: 'text' },
    { field: 'summary', label: 'Rezumat', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista sedinte pentru filtrele curente.',
  detailChildren: [MEETING_PARTICIPANTS_CHILD, MEETING_DOCUMENTS_CHILD, MEETING_VOTES_CHILD, MEETING_MINUTES_CHILD, MEETING_RESOLUTIONS_CHILD],
};

const CP_MEETINGS_RESOURCE: EducationResourceConfig = {
  ...MEETINGS_RESOURCE,
  key: 'cp_meetings',
  label: 'CP complet',
  description: 'Sedinte, documente oficiale si trasee procedurale pentru Consiliul Profesoral.',
  endpoint: '/api/education/governance/meetings?filter.organism=cp',
  createWizardRoute: '/education/governance/ca-wizard?organism=cp',
  columns: [
    { field: 'school_year', header: 'An scolar', sortable: true, filter: 'text', width: '9rem' },
    { field: 'organism', header: 'Organism', type: 'tag', sortable: true, filter: 'select', options: ORGANISM_OPTIONS, width: '10rem' },
    { field: 'title', header: 'Titlu', sortable: true, filter: 'text', width: '24rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: MEETING_STATUS_OPTIONS, width: '10rem' },
    { field: 'meeting_date', header: 'Data', sortable: true, width: '9rem' },
    { field: 'participants_count', header: 'Participanti', type: 'number', sortable: true, width: '8rem' },
  ],
};

const GOVERNANCE_MEMBERSHIPS_RESOURCE: EducationResourceConfig = {
  key: 'governance_memberships',
  label: 'Componenta organisme',
  icon: 'pi pi-sitemap',
  endpoint: '/api/education/governance/memberships',
  createEndpoint: '/api/education/governance/memberships',
  readPermission: 'education.governance.read',
  managePermission: 'education.governance.manage',
  description: 'Mandate, roluri, drept de vot si situatie activa pentru CA, CP, CEAC si CFDCD.',
  columns: [
    { field: 'school_year', header: 'An scolar', sortable: true, filter: 'text', width: '9rem' },
    { field: 'organism', header: 'Organism', type: 'tag', sortable: true, filter: 'select', options: ORGANISM_OPTIONS, width: '10rem' },
    { field: 'full_name', header: 'Nume', sortable: true, filter: 'text', width: '18rem' },
    { field: 'role_name', header: 'Rol', sortable: true, filter: 'text', width: '16rem' },
    { field: 'mandate_from', header: 'Mandat de la', sortable: true, width: '10rem' },
    { field: 'mandate_to', header: 'Mandat pana la', sortable: true, width: '10rem' },
    { field: 'voting_right', header: 'Vot', type: 'boolean', sortable: true, width: '7rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: GOVERNANCE_MEMBERSHIP_STATUS_OPTIONS, width: '10rem' },
  ],
  createFields: [
    { field: 'school_year', label: 'An scolar', type: 'text', defaultValue: '2025-2026', required: true },
    { field: 'organism', label: 'Organism', type: 'select', options: ORGANISM_OPTIONS, defaultValue: 'ca', required: true },
    { field: 'full_name', label: 'Nume complet', type: 'text', wide: true, required: true },
    { field: 'role_name', label: 'Rol / functie', type: 'text', required: true },
    { field: 'mandate_from', label: 'Mandat de la', type: 'date', required: true },
    { field: 'mandate_to', label: 'Mandat pana la', type: 'date', required: true },
    { field: 'voting_right', label: 'Are drept de vot', type: 'boolean', defaultValue: true },
    { field: 'status', label: 'Status mandat', type: 'select', options: GOVERNANCE_MEMBERSHIP_STATUS_OPTIONS, defaultValue: 'activ', required: true },
    { field: 'notes', label: 'Observatii', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista membri configurati pentru organismele de guvernanta.',
};

const GOVERNANCE_BODY_MEMBERSHIPS_CHILD: EducationDetailChildResourceConfig = {
  key: 'body_memberships',
  label: 'Membri organism',
  icon: 'pi pi-users',
  description: 'Membrii activi si istoricul mandatelor pentru organismul selectat.',
  listEndpoint: (parentRow) => `/api/education/governance/memberships?filter.school_year=${encodeURIComponent(String(parentRow['school_year'] ?? ''))}&filter.organism=${encodeURIComponent(String(parentRow['organism'] ?? ''))}`,
  detailEndpoint: (_parentRow, childRow) => `/api/education/governance/memberships/${childRow['id']}`,
  createEndpoint: () => '/api/education/governance/memberships',
  allowCreate: false,
  readPermission: 'education.governance.read',
  managePermission: 'education.governance.manage',
  columns: GOVERNANCE_MEMBERSHIPS_RESOURCE.columns,
  createFields: GOVERNANCE_MEMBERSHIPS_RESOURCE.createFields,
  emptyText: 'Nu exista membri pentru organismul selectat.',
};

const GOVERNANCE_BODY_MEETINGS_CHILD: EducationDetailChildResourceConfig = {
  key: 'body_meetings',
  label: 'Sedinte organism',
  icon: 'pi pi-calendar',
  description: 'Sedintele asociate organismului selectat.',
  listEndpoint: (parentRow) => `/api/education/governance/meetings?filter.school_year=${encodeURIComponent(String(parentRow['school_year'] ?? ''))}&filter.organism=${encodeURIComponent(String(parentRow['organism'] ?? ''))}`,
  detailEndpoint: (_parentRow, childRow) => `/api/education/governance/meetings/${childRow['id']}`,
  createEndpoint: () => '/api/education/governance/meetings',
  allowCreate: false,
  readPermission: 'education.governance.read',
  managePermission: 'education.governance.manage',
  columns: MEETINGS_RESOURCE.columns,
  createFields: MEETINGS_RESOURCE.createFields,
  emptyText: 'Nu exista sedinte pentru organismul selectat.',
};

const GOVERNANCE_BODIES_RESOURCE: EducationResourceConfig = {
  key: 'governance_bodies',
  label: 'CA / CP / CEAC',
  icon: 'pi pi-building-columns',
  endpoint: '/api/education/governance/bodies',
  createEndpoint: '',
  allowCreate: false,
  detailSummaryKind: 'governance-body-completeness',
  detailSummaryEndpoint: (row) => `/api/education/governance/bodies/${String(row['id'] ?? '')}/completeness-summary`,
  readPermission: 'education.governance.read',
  description: 'Constituire si operabilitate pentru Consiliul de Administratie, Consiliul Profesoral si celelalte organisme.',
  columns: [
    { field: 'school_year', header: 'An scolar', sortable: true, filter: 'text', width: '9rem' },
    { field: 'organism', header: 'Organism', type: 'tag', sortable: true, filter: 'select', options: ORGANISM_OPTIONS, width: '10rem' },
    { field: 'active_members', header: 'Membri activi', type: 'number', sortable: true, width: '8rem' },
    { field: 'voting_members', header: 'Membri cu vot', type: 'number', sortable: true, width: '8rem' },
    { field: 'held_meetings', header: 'Sedinte tinute', type: 'number', sortable: true, width: '8rem' },
    { field: 'latest_meeting_on', header: 'Ultima sedinta', sortable: true, width: '9rem' },
    { field: 'readiness_status', header: 'Stare', type: 'tag', sortable: true, width: '9rem' },
  ],
  createFields: [],
  detailChildren: [GOVERNANCE_BODY_MEMBERSHIPS_CHILD, GOVERNANCE_BODY_MEETINGS_CHILD],
  emptyText: 'Nu exista organisme constituite pentru filtrele curente.',
};

const COMMITTEE_MEMBERS_CHILD: EducationDetailChildResourceConfig = {
  key: 'committee_members',
  label: 'Membri comisie',
  icon: 'pi pi-users',
  description: 'Componenta nominala, roluri si drept de vot pentru comisia selectata.',
  listEndpoint: (parentRow) => `/api/education/committees/records/${parentRow['id']}/members`,
  detailEndpoint: (parentRow, childRow) => `/api/education/committees/records/${parentRow['id']}/members/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/committees/records/${parentRow['id']}/members`,
  readPermission: 'education.governance.read',
  managePermission: 'education.governance.manage',
  columns: [
    { field: 'full_name', header: 'Nume', sortable: true, filter: 'text', width: '18rem' },
    { field: 'role_name', header: 'Rol', sortable: true, filter: 'text', width: '14rem' },
    { field: 'member_type', header: 'Tip membru', type: 'tag', sortable: true, filter: 'select', options: COMMITTEE_MEMBER_TYPE_OPTIONS, width: '12rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: COMMITTEE_MEMBER_STATUS_OPTIONS, width: '10rem' },
    { field: 'voting_right', header: 'Vot', type: 'boolean', sortable: true, width: '7rem' },
    { field: 'appointed_on', header: 'Numit la', sortable: true, width: '9rem' },
  ],
  createFields: [
    { field: 'full_name', label: 'Nume complet', type: 'text', wide: true, required: true },
    { field: 'role_name', label: 'Rol / functie', type: 'text', required: true },
    { field: 'member_type', label: 'Tip membru', type: 'select', options: COMMITTEE_MEMBER_TYPE_OPTIONS, defaultValue: 'membru', required: true },
    { field: 'status', label: 'Status', type: 'select', options: COMMITTEE_MEMBER_STATUS_OPTIONS, defaultValue: 'active', required: true },
    { field: 'voting_right', label: 'Are drept de vot', type: 'boolean', defaultValue: true },
    { field: 'appointed_on', label: 'Numit la', type: 'date', required: true },
    { field: 'released_on', label: 'Eliberat la', type: 'date' },
    { field: 'notes', label: 'Observatii', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista membri pentru comisia selectata.',
};

const COMMITTEES_RESOURCE: EducationResourceConfig = {
  key: 'committees',
  label: 'Comisii',
  icon: 'pi pi-briefcase',
  endpoint: '/api/education/committees/records',
  createEndpoint: '/api/education/committees/records',
  detailSummaryKind: 'committee-completeness',
  detailSummaryEndpoint: (row) => `/api/education/committees/records/${String(row['id'] ?? '')}/completeness-summary`,
  readPermission: 'education.governance.read',
  managePermission: 'education.governance.manage',
  description: 'Comisii institutionale si comisia de evaluare a personalului didactic.',
  columns: [
    { field: 'committee_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'school_year', header: 'An', sortable: true, filter: 'text', width: '8rem' },
    { field: 'committee_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: COMMITTEE_TYPE_OPTIONS, width: '14rem' },
    { field: 'title', header: 'Titlu', sortable: true, filter: 'text', width: '22rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: COMMITTEE_STATUS_OPTIONS, width: '10rem' },
    { field: 'starts_on', header: 'De la', sortable: true, width: '9rem' },
    { field: 'evaluation_scope', header: 'Evaluare', type: 'boolean', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'school_year', label: 'An scolar', type: 'text', defaultValue: '2025-2026', required: true },
    { field: 'committee_type', label: 'Tip comisie', type: 'select', options: COMMITTEE_TYPE_OPTIONS, defaultValue: 'evaluare_personal_didactic', required: true },
    { field: 'title', label: 'Titlu', type: 'text', wide: true, required: true },
    { field: 'status', label: 'Status', type: 'select', options: COMMITTEE_STATUS_OPTIONS, defaultValue: 'active', required: true },
    { field: 'decision_reference', label: 'Decizie / referinta', type: 'text' },
    { field: 'starts_on', label: 'Data inceput', type: 'date', required: true },
    { field: 'ends_on', label: 'Data finalizare', type: 'date' },
    { field: 'evaluation_scope', label: 'Acopera evaluarea', type: 'boolean', defaultValue: false },
    { field: 'notes', label: 'Observatii', type: 'textarea', wide: true },
  ],
  detailChildren: [COMMITTEE_MEMBERS_CHILD],
  emptyText: 'Nu exista comisii pentru filtrele curente.',
};

const DECISION_ISSUANCES_CHILD: EducationDetailChildResourceConfig = {
  key: 'decision_issuances',
  label: 'Emitere si comunicare',
  icon: 'pi pi-send',
  description: 'Exemplare semnate, comunicari interne si confirmari de primire pentru decizia selectata.',
  listEndpoint: (parentRow) => `/api/education/decisions/records/${parentRow['id']}/issuances`,
  detailEndpoint: (parentRow, childRow) => `/api/education/decisions/records/${parentRow['id']}/issuances/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/decisions/records/${parentRow['id']}/issuances`,
  readPermission: 'education.decisions.issuance.read',
  managePermission: 'education.decisions.issuance.manage',
  columns: [
    { field: 'issuance_code', header: 'Cod', sortable: true, filter: 'text', width: '12rem' },
    { field: 'document_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: DECISION_ISSUANCE_TYPE_OPTIONS, width: '12rem' },
    { field: 'recipient_name', header: 'Destinatar', sortable: true, filter: 'text', width: '18rem' },
    { field: 'recipient_role', header: 'Rol / compartiment', sortable: true, filter: 'text', width: '16rem' },
    { field: 'delivery_channel', header: 'Canal', type: 'tag', sortable: true, filter: 'select', options: DECISION_ISSUANCE_CHANNEL_OPTIONS, width: '10rem' },
    { field: 'delivery_status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: DECISION_ISSUANCE_STATUS_OPTIONS, width: '10rem' },
    { field: 'signed_on', header: 'Semnat la', sortable: true, width: '9rem' },
    { field: 'delivered_on', header: 'Predat la', sortable: true, width: '9rem' },
    { field: 'acknowledged_on', header: 'Confirmat la', sortable: true, width: '10rem' },
  ],
  createFields: [
    { field: 'document_type', label: 'Tip document emis', type: 'select', options: DECISION_ISSUANCE_TYPE_OPTIONS, defaultValue: 'decizie', required: true },
    { field: 'recipient_name', label: 'Destinatar', type: 'text', wide: true, required: true },
    { field: 'recipient_role', label: 'Rol / compartiment', type: 'text' },
    { field: 'delivery_channel', label: 'Canal transmitere', type: 'select', options: DECISION_ISSUANCE_CHANNEL_OPTIONS, defaultValue: 'registratura', required: true },
    { field: 'delivery_status', label: 'Status emitere', type: 'select', options: DECISION_ISSUANCE_STATUS_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'signed_on', label: 'Data semnarii', type: 'date' },
    { field: 'delivered_on', label: 'Data predarii / transmiterii', type: 'date' },
    { field: 'acknowledged_on', label: 'Data confirmarii', type: 'date' },
    { field: 'file_reference', label: 'Referinta fisier / registratura', type: 'text' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista emitere sau comunicari pentru decizia selectata.',
};

const DECISION_PUBLICATION_STEPS_CHILD: EducationDetailChildResourceConfig = {
  key: 'decision_publication_steps',
  label: 'Flux publicare',
  icon: 'pi pi-megaphone',
  description: 'Analiza juridica, anonimizare, aprobare si publicare controlata pentru decizia selectata.',
  listEndpoint: (parentRow) => `/api/education/decisions/records/${parentRow['id']}/publication-steps`,
  detailEndpoint: (parentRow, childRow) => `/api/education/decisions/records/${parentRow['id']}/publication-steps/${childRow['id']}`,
  createEndpoint: (parentRow) => `/api/education/decisions/records/${parentRow['id']}/publication-steps`,
  readPermission: 'education.compliance.read',
  managePermission: 'education.compliance.manage',
  columns: [
    { field: 'step_order', header: 'Ordine', type: 'number', sortable: true, width: '7rem' },
    { field: 'step_type', header: 'Etapa', type: 'tag', sortable: true, filter: 'select', options: DECISION_PUBLICATION_STEP_OPTIONS, width: '14rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: DECISION_PUBLICATION_STEP_STATUS_OPTIONS, width: '10rem' },
    { field: 'responsible_name', header: 'Responsabil', sortable: true, filter: 'text', width: '16rem' },
    { field: 'publication_channel', header: 'Canal', type: 'tag', sortable: true, filter: 'select', options: PUBLICATION_CHANNEL_OPTIONS, width: '10rem' },
    { field: 'due_on', header: 'Termen', sortable: true, width: '9rem' },
    { field: 'completed_on', header: 'Finalizat', sortable: true, width: '9rem' },
    { field: 'publication_reference', header: 'Referinta', sortable: true, filter: 'text', width: '14rem' },
  ],
  createFields: [
    { field: 'step_order', label: 'Ordine etapa', type: 'number', defaultValue: 1, required: true },
    { field: 'step_type', label: 'Etapa flux', type: 'select', options: DECISION_PUBLICATION_STEP_OPTIONS, defaultValue: 'analiza_juridica', required: true },
    { field: 'status', label: 'Status etapa', type: 'select', options: DECISION_PUBLICATION_STEP_STATUS_OPTIONS, defaultValue: 'pending', required: true },
    { field: 'responsible_name', label: 'Responsabil', type: 'text', wide: true, required: true },
    { field: 'publication_channel', label: 'Canal publicare', type: 'select', options: PUBLICATION_CHANNEL_OPTIONS, defaultValue: 'site_public' },
    { field: 'due_on', label: 'Termen limita', type: 'date', required: true },
    { field: 'completed_on', label: 'Data finalizarii', type: 'date' },
    { field: 'publication_reference', label: 'Referinta aviz / publicare', type: 'text' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista etape de publicare pentru decizia selectata.',
};

const PUBLICATIONS_RESOURCE: EducationResourceConfig = {
  key: 'publications',
  label: 'Publicare si anonimizare',
  icon: 'pi pi-megaphone',
  endpoint: '/api/education/compliance/publications',
  createEndpoint: '/api/education/compliance/publications',
  readPermission: 'education.compliance.read',
  managePermission: 'education.compliance.manage',
  description: 'Registru pentru documente supuse publicarii, status de anonimizare si canal de publicare.',
  columns: [
    { field: 'publication_code', header: 'Cod', sortable: true, filter: 'text', width: '11rem' },
    { field: 'domain', header: 'Domeniu', type: 'tag', sortable: true, filter: 'select', options: PUBLICATION_DOMAIN_OPTIONS, width: '12rem' },
    { field: 'entity_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: PUBLICATION_ENTITY_TYPE_OPTIONS, width: '14rem' },
    { field: 'entity_label', header: 'Document / entitate', sortable: true, filter: 'text', width: '24rem' },
    { field: 'publication_channel', header: 'Canal', type: 'tag', sortable: true, filter: 'select', options: PUBLICATION_CHANNEL_OPTIONS, width: '11rem' },
    { field: 'publication_status', header: 'Publicare', type: 'tag', sortable: true, filter: 'select', options: PUBLICATION_STATUS_OPTIONS, width: '10rem' },
    { field: 'anonymization_status', header: 'Anonimizare', type: 'tag', sortable: true, filter: 'select', options: GOVERNANCE_RESOLUTION_ANONYMIZATION_OPTIONS, width: '12rem' },
    { field: 'published_on', header: 'Data', sortable: true, width: '9rem' },
  ],
  createFields: [
    { field: 'domain', label: 'Domeniu', type: 'select', options: PUBLICATION_DOMAIN_OPTIONS, defaultValue: 'guvernanta', required: true },
    { field: 'entity_type', label: 'Tip entitate', type: 'select', options: PUBLICATION_ENTITY_TYPE_OPTIONS, defaultValue: 'hotarare', required: true },
    { field: 'entity_label', label: 'Denumire document / entitate', type: 'text', wide: true, required: true },
    { field: 'publication_channel', label: 'Canal publicare', type: 'select', options: PUBLICATION_CHANNEL_OPTIONS, defaultValue: 'site_public', required: true },
    { field: 'publication_status', label: 'Status publicare', type: 'select', options: PUBLICATION_STATUS_OPTIONS, defaultValue: 'pregatit', required: true },
    { field: 'anonymization_status', label: 'Status anonimizare', type: 'select', options: GOVERNANCE_RESOLUTION_ANONYMIZATION_OPTIONS, defaultValue: 'necesara', required: true },
    { field: 'mandatory', label: 'Publicare obligatorie', type: 'boolean', defaultValue: true },
    { field: 'published_on', label: 'Data publicarii', type: 'date' },
    { field: 'reviewed_by', label: 'Verificat de', type: 'text' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista inregistrari de publicare pentru filtrele curente.',
};

const DECISIONS_RESOURCE: EducationResourceConfig = {
  key: 'decisions',
  label: 'Decizii CA/CP',
  icon: 'pi pi-verified',
  endpoint: '/api/education/decisions/records',
  createEndpoint: '/api/education/decisions/records',
  readPermission: 'education.decisions.read',
  managePermission: 'education.decisions.manage',
  description: 'Decizii cu baza legala, semnare, publicare si status de anonimizare.',
  columns: [
    { field: 'decision_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
    { field: 'school_year', header: 'An', sortable: true, filter: 'text', width: '8rem' },
    { field: 'organism', header: 'Organism', type: 'tag', sortable: true, filter: 'select', options: ORGANISM_OPTIONS, width: '9rem' },
    { field: 'title', header: 'Titlu', sortable: true, filter: 'text', width: '22rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: DECISION_STATUS_OPTIONS, width: '10rem' },
    { field: 'publication_status', header: 'Publicare', type: 'tag', sortable: true, filter: 'select', options: DECISION_PUBLICATION_OPTIONS, width: '14rem' },
    { field: 'decision_date', header: 'Data', sortable: true, width: '9rem' },
  ],
  createFields: [
    { field: 'school_year', label: 'An scolar', type: 'text', defaultValue: '2025-2026', required: true },
    { field: 'organism', label: 'Organism', type: 'select', options: ORGANISM_OPTIONS, defaultValue: 'ca', required: true },
    { field: 'title', label: 'Titlu decizie', type: 'text', wide: true, required: true },
    { field: 'status', label: 'Status', type: 'select', options: DECISION_STATUS_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'publication_status', label: 'Status publicare', type: 'select', options: DECISION_PUBLICATION_OPTIONS, defaultValue: 'internal', required: true },
    { field: 'decision_date', label: 'Data deciziei', type: 'date' },
    { field: 'legal_basis', label: 'Baza legala', type: 'textarea', wide: true },
    { field: 'signed_by', label: 'Semnat de', type: 'text' },
    { field: 'summary', label: 'Rezumat', type: 'textarea', wide: true },
  ],
  detailChildren: [DECISION_ISSUANCES_CHILD, DECISION_PUBLICATION_STEPS_CHILD],
  emptyText: 'Nu exista decizii pentru filtrele curente.',
};

const MANAGERIAL_RESOURCE: EducationResourceConfig = {
  key: 'managerial',
  label: 'Dosare manageriale',
  icon: 'pi pi-briefcase',
  endpoint: '/api/education/managerial/records',
  createEndpoint: '/api/education/managerial/records',
  createWizardRoute: '/education/governance/managerial-wizard',
  detailSummaryKind: 'managerial-portfolio',
  detailSummaryEndpoint: (row) => `/api/education/managerial/records/${String(row['id'] ?? '')}/portfolio-summary`,
  pdfEndpoint: (row) => `/api/education/managerial/records/${row['id']}/pdf`,
  pdfFilename: (row) => `dosar-managerial-${String(row['dossier_code'] ?? row['id'] ?? 'managerial')}.pdf`,
  pdfActionLabel: 'Dosar PDF',
  readPermission: 'education.managerial.read',
  managePermission: 'education.managerial.manage',
  description: 'PDI/PAS, plan anual, RAEI, organigrama, incadrare, orar si portofoliile manageriale ale conducerii.',
  columns: [
    { field: 'dossier_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
    { field: 'school_year', header: 'An', sortable: true, filter: 'text', width: '8rem' },
    { field: 'dossier_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: MANAGERIAL_TYPE_OPTIONS, width: '14rem' },
    { field: 'title', header: 'Titlu', sortable: true, filter: 'text', width: '22rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: MANAGERIAL_STATUS_OPTIONS, width: '10rem' },
    { field: 'owner_name', header: 'Responsabil', sortable: true, filter: 'text', width: '12rem' },
    { field: 'due_on', header: 'Termen', sortable: true, width: '9rem' },
  ],
  createFields: [
    { field: 'school_year', label: 'An scolar', type: 'text', defaultValue: '2025-2026', required: true },
    { field: 'dossier_type', label: 'Tip dosar', type: 'select', options: MANAGERIAL_TYPE_OPTIONS, defaultValue: 'annual_plan', required: true },
    { field: 'title', label: 'Titlu', type: 'text', wide: true, required: true },
    { field: 'status', label: 'Status', type: 'select', options: MANAGERIAL_STATUS_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'owner_name', label: 'Responsabil', type: 'text' },
    { field: 'due_on', label: 'Termen', type: 'date' },
    { field: 'publication_required', label: 'Publicare necesara', type: 'boolean', defaultValue: false },
    { field: 'summary', label: 'Rezumat', type: 'textarea', wide: true },
  ],
  detailChildren: [MANAGERIAL_DOCUMENTS_CHILD, MANAGERIAL_WORKFLOW_CHILD],
  emptyText: 'Nu exista dosare manageriale pentru filtrele curente.',
};

const DIRECTOR_PORTFOLIO_RESOURCE: EducationResourceConfig = {
  ...MANAGERIAL_RESOURCE,
  key: 'director_portfolio',
  label: 'Portofoliu director',
  endpoint: '/api/education/managerial/records?filter.dossier_type=director_portfolio',
  createWizardRoute: '/education/governance/managerial-wizard?dossierType=director_portfolio',
  description: 'Portofoliul directorului cu documente, workflow, avizare si publicare urmarite separat.',
};

const ADJUNCT_DIRECTOR_PORTFOLIO_RESOURCE: EducationResourceConfig = {
  ...MANAGERIAL_RESOURCE,
  key: 'adjunct_director_portfolio',
  label: 'Portofoliu director adjunct',
  endpoint: '/api/education/managerial/records?filter.dossier_type=adjunct_director_portfolio',
  createWizardRoute: '/education/governance/managerial-wizard?dossierType=adjunct_director_portfolio',
  description: 'Portofoliul directorului adjunct cu documente, workflow, avizare si publicare urmarite separat.',
};

const REGULATIONS_RESOURCE: EducationResourceConfig = {
  key: 'regulations',
  label: 'ROF / ROI',
  icon: 'pi pi-book',
  endpoint: '/api/education/regulations/records',
  createEndpoint: '/api/education/regulations/records',
  detailSummaryKind: 'regulation-procedural',
  detailSummaryEndpoint: (row) => `/api/education/regulations/records/${String(row['id'] ?? '')}/procedural-summary`,
  readPermission: 'education.regulations.read',
  managePermission: 'education.regulations.manage',
  description: 'Regulamente, consultare, aprobare, publicare si revizuire periodica.',
  columns: [
    { field: 'regulation_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
    { field: 'school_year', header: 'An', sortable: true, filter: 'text', width: '8rem' },
    { field: 'regulation_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: REGULATION_TYPE_OPTIONS, width: '10rem' },
    { field: 'title', header: 'Titlu', sortable: true, filter: 'text', width: '22rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: REGULATION_STATUS_OPTIONS, width: '12rem' },
    { field: 'approval_status', header: 'Aprobare', type: 'tag', sortable: true, filter: 'select', options: REGULATION_APPROVAL_OPTIONS, width: '12rem' },
    { field: 'review_due_on', header: 'Revizuire', sortable: true, width: '9rem' },
  ],
  createFields: [
    { field: 'school_year', label: 'An scolar', type: 'text', defaultValue: '2025-2026', required: true },
    { field: 'regulation_type', label: 'Tip regulament', type: 'select', options: REGULATION_TYPE_OPTIONS, defaultValue: 'rof', required: true },
    { field: 'title', label: 'Titlu', type: 'text', wide: true, required: true },
    { field: 'status', label: 'Status', type: 'select', options: REGULATION_STATUS_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'approval_status', label: 'Status aprobare', type: 'select', options: REGULATION_APPROVAL_OPTIONS, defaultValue: 'working_group', required: true },
    { field: 'owner_name', label: 'Responsabil', type: 'text' },
    { field: 'review_due_on', label: 'Revizuire pana la', type: 'date' },
    { field: 'approved_on', label: 'Aprobat la', type: 'date' },
    { field: 'summary', label: 'Rezumat', type: 'textarea', wide: true },
  ],
  detailChildren: [REGULATION_VERSIONS_CHILD, REGULATION_WORKFLOW_CHILD],
  emptyText: 'Nu exista regulamente pentru filtrele curente.',
};

const PERSONNEL_RESOURCE: EducationResourceConfig = {
  key: 'personnel',
  label: 'Cadre didactice',
  icon: 'pi pi-id-card',
  endpoint: '/api/education/personnel/records',
  createEndpoint: '/api/education/personnel/records',
  createWizardRoute: '/education/personnel/wizard',
  detailSummaryKind: 'personnel-portfolio-dossier',
  detailSummaryEndpoint: (row) => `/api/education/personnel/records/${String(row['id'] ?? '')}/portfolio-dossier-summary`,
  readPermission: 'education.personnel.read',
  managePermission: 'education.personnel.manage',
  description: 'Fise de personal, incadrare, statut, evaluare, mobilitate si portofoliu.',
  columns: [
    { field: 'employee_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
    { field: 'full_name', header: 'Nume', sortable: true, filter: 'text', width: '18rem' },
    { field: 'role_title', header: 'Functie', sortable: true, filter: 'text', width: '14rem' },
    { field: 'employment_type', header: 'Incadrare', type: 'tag', sortable: true, filter: 'select', options: PERSONNEL_EMPLOYMENT_OPTIONS, width: '12rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: PERSONNEL_STATUS_OPTIONS, width: '10rem' },
    { field: 'evaluation_status', header: 'Evaluare', type: 'tag', sortable: true, filter: 'select', options: PERSONNEL_EVALUATION_STATUS_OPTIONS, width: '10rem' },
    { field: 'has_portfolio', header: 'Portofoliu', type: 'boolean', sortable: true, width: '8rem' },
  ],
  createFields: [
    { field: 'full_name', label: 'Nume complet', type: 'text', wide: true, required: true },
    { field: 'role_title', label: 'Functie', type: 'text', required: true },
    { field: 'employment_type', label: 'Incadrare', type: 'select', options: PERSONNEL_EMPLOYMENT_OPTIONS, defaultValue: 'titular', required: true },
    { field: 'status', label: 'Status', type: 'select', options: PERSONNEL_STATUS_OPTIONS, defaultValue: 'active', required: true },
    { field: 'evaluation_status', label: 'Evaluare', type: 'select', options: PERSONNEL_EVALUATION_STATUS_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'mobility_stage', label: 'Etapa mobilitate', type: 'select', options: PERSONNEL_MOBILITY_STAGE_OPTIONS, defaultValue: 'none', required: true },
    { field: 'school_year', label: 'An scolar', type: 'text', defaultValue: '2025-2026', required: true },
    { field: 'assigned_unit', label: 'Structura', type: 'text' },
    { field: 'phone', label: 'Telefon', type: 'text' },
    { field: 'email', label: 'Email', type: 'text' },
    { field: 'has_portfolio', label: 'Are portofoliu', type: 'boolean', defaultValue: false },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  detailChildren: [PERSONNEL_ASSIGNMENTS_CHILD, PERSONNEL_DISCIPLINARY_CASES_CHILD, PERSONNEL_FILE_DOCUMENTS_CHILD, PERSONNEL_ACCESS_EVENTS_CHILD],
  emptyText: 'Nu exista cadre didactice pentru filtrele curente.',
};

const EVALUATIONS_RESOURCE: EducationResourceConfig = {
  key: 'evaluations',
  label: 'Evaluari anuale',
  icon: 'pi pi-list',
  endpoint: '/api/education/evaluations/records',
  createEndpoint: '/api/education/evaluations/records',
  createWizardRoute: '/education/personnel/evaluations-wizard',
  readPermission: 'education.evaluations.read',
  managePermission: 'education.evaluations.manage',
  pdfEndpoint: (row) => `/api/education/evaluations/records/${row['id']}/pdf`,
  pdfFilename: (row) => `fisa-evaluare-${String(row['evaluation_code'] ?? row['id'] ?? 'evaluare')}.pdf`,
  pdfActionLabel: 'Fisa PDF',
  description: 'Punctaj, calificativ, contestare, comunicare rezultat si finalizare.',
  columns: [
    { field: 'evaluation_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
    { field: 'full_name', header: 'Nume', sortable: true, filter: 'text', width: '18rem' },
    { field: 'school_year', header: 'An', sortable: true, filter: 'text', width: '8rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: EVALUATION_STATUS_OPTIONS, width: '10rem' },
    { field: 'qualification', header: 'Calificativ', type: 'tag', sortable: true, filter: 'select', options: EVALUATION_QUALIFICATION_OPTIONS, width: '11rem' },
    { field: 'score', header: 'Punctaj', type: 'number', sortable: true, width: '8rem' },
    { field: 'finalized_on', header: 'Finalizata', sortable: true, width: '9rem' },
  ],
  createFields: [
    { field: 'employee_code', label: 'Cod angajat', type: 'text', required: true },
    { field: 'full_name', label: 'Nume complet', type: 'text', wide: true, required: true },
    { field: 'role_title', label: 'Functie', type: 'text', required: true },
    { field: 'school_year', label: 'An scolar', type: 'text', defaultValue: '2025-2026', required: true },
    { field: 'status', label: 'Status', type: 'select', options: EVALUATION_STATUS_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'score', label: 'Punctaj (0-100)', type: 'number', defaultValue: 0, min: 0, max: 100, step: 0.01 },
    { field: 'evaluator_name', label: 'Evaluator', type: 'text' },
    { field: 'finalized_on', label: 'Data finalizarii', type: 'date' },
    { field: 'summary', label: 'Rezumat', type: 'textarea', wide: true },
  ],
  detailChildren: [EVALUATION_SELF_REVIEWS_CHILD, EVALUATION_CRITERIA_CHILD, EVALUATION_APPEALS_CHILD, EVALUATION_RESULT_ISSUES_CHILD],
  emptyText: 'Nu exista evaluari pentru filtrele curente.',
};

const DECLARATIONS_RESOURCE: EducationResourceConfig = {
  key: 'declarations',
  label: 'Declaratii',
  icon: 'pi pi-list',
  endpoint: '/api/education/declarations/records',
  createEndpoint: '/api/education/declarations/records',
  createWizardRoute: '/education/personnel/declarations-wizard',
  readPermission: 'education.declarations.read',
  managePermission: 'education.declarations.manage',
  description: 'Declaratii si adeverinte asociate cadrului didactic.',
  columns: [
    { field: 'declaration_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
    { field: 'full_name', header: 'Nume', sortable: true, filter: 'text', width: '18rem' },
    { field: 'school_year', header: 'An', sortable: true, filter: 'text', width: '8rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: DECLARATION_STATUS_OPTIONS, width: '10rem' },
    { field: 'declaration_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: DECLARATION_TYPE_OPTIONS, width: '16rem' },
    { field: 'submitted_on', header: 'Data', sortable: true, width: '9rem' },
  ],
  createFields: [
    { field: 'employee_code', label: 'Cod angajat', type: 'text', required: true },
    { field: 'full_name', label: 'Nume complet', type: 'text', wide: true, required: true },
    { field: 'declaration_type', label: 'Tip declaratie', type: 'select', options: DECLARATION_TYPE_OPTIONS, defaultValue: 'authenticity', required: true },
    { field: 'status', label: 'Status', type: 'select', options: DECLARATION_STATUS_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'school_year', label: 'An scolar', type: 'text', defaultValue: '2025-2026', required: true },
    { field: 'submitted_on', label: 'Data depunerii', type: 'date' },
    { field: 'valid_until', label: 'Valabila pana la', type: 'date' },
    { field: 'summary', label: 'Rezumat', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista declaratii pentru filtrele curente.',
};

const MOBILITY_RESOURCE: EducationResourceConfig = {
  key: 'mobility',
  label: 'Mobilitate',
  icon: 'pi pi-list',
  endpoint: '/api/education/mobility/records',
  createEndpoint: '/api/education/mobility/records',
  createWizardRoute: '/education/personnel/mobility-wizard',
  pdfEndpoint: (row) => `/api/education/mobility/records/${row['id']}/pdf`,
  pdfFilename: (row) => `mobilitate-${row['case_code'] ?? row['id']}.pdf`,
  pdfActionLabel: 'Dosar PDF',
  readPermission: 'education.mobility.read',
  managePermission: 'education.mobility.manage',
  description: 'Cazuri de mobilitate, transfer, etape, sursa si destinatie.',
  columns: [
    { field: 'case_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
    { field: 'full_name', header: 'Nume', sortable: true, filter: 'text', width: '18rem' },
    { field: 'school_year', header: 'An', sortable: true, filter: 'text', width: '8rem' },
    { field: 'stage', header: 'Etapa', type: 'tag', sortable: true, filter: 'select', options: MOBILITY_STAGE_OPTIONS, width: '10rem' },
    { field: 'request_type', header: 'Tip', type: 'tag', sortable: true, filter: 'select', options: MOBILITY_REQUEST_TYPE_OPTIONS, width: '12rem' },
    { field: 'submitted_on', header: 'Data', sortable: true, width: '9rem' },
  ],
  createFields: [
    { field: 'employee_code', label: 'Cod angajat', type: 'text', required: true },
    { field: 'full_name', label: 'Nume complet', type: 'text', wide: true, required: true },
    { field: 'school_year', label: 'An scolar', type: 'text', defaultValue: '2025-2026', required: true },
    { field: 'request_type', label: 'Tip solicitare', type: 'select', options: MOBILITY_REQUEST_TYPE_OPTIONS, defaultValue: 'transfer', required: true },
    { field: 'stage', label: 'Etapa', type: 'select', options: MOBILITY_STAGE_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'status', label: 'Status', type: 'select', options: MOBILITY_STATUS_OPTIONS, defaultValue: 'open', required: true },
    { field: 'source_school', label: 'Unitate sursa', type: 'text' },
    { field: 'destination_school', label: 'Unitate destinatie', type: 'text' },
    { field: 'submitted_on', label: 'Data depunerii', type: 'date' },
    { field: 'reviewed_by', label: 'Analizat de', type: 'text' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  detailChildren: [MOBILITY_DOCUMENTS_CHILD, MOBILITY_SCORES_CHILD, MOBILITY_APPEALS_CHILD, MOBILITY_FINAL_DECISIONS_CHILD, MOBILITY_RESULT_ISSUES_CHILD],
  emptyText: 'Nu exista inregistrari pentru mobilitate.',
};

const MERIT_RESOURCE: EducationResourceConfig = {
  key: 'gradatii',
  label: 'Gradatii',
  icon: 'pi pi-list',
  endpoint: '/api/education/gradatii/records',
  createEndpoint: '/api/education/gradatii/records',
  createWizardRoute: '/education/personnel/merit-wizard',
  pdfEndpoint: (row) => `/api/education/gradatii/records/${row['id']}/pdf`,
  pdfFilename: (row) => `gradatie-merit-${row['grant_code'] ?? row['id']}.pdf`,
  pdfActionLabel: 'Dosar PDF',
  readPermission: 'education.gradatii.read',
  managePermission: 'education.gradatii.manage',
  description: 'Gradatii de merit, punctaj, comisie si finantare.',
  columns: [
    { field: 'grant_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
    { field: 'full_name', header: 'Nume', sortable: true, filter: 'text', width: '18rem' },
    { field: 'school_year', header: 'An', sortable: true, filter: 'text', width: '8rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: MERIT_STATUS_OPTIONS, width: '10rem' },
    { field: 'score', header: 'Punctaj', type: 'number', sortable: true, width: '8rem' },
    { field: 'decision_date', header: 'Data', sortable: true, width: '9rem' },
  ],
  createFields: [
    { field: 'full_name', label: 'Nume complet', type: 'text', wide: true, required: true },
    { field: 'role_title', label: 'Functie', type: 'text', required: true },
    { field: 'school_year', label: 'An scolar', type: 'text', defaultValue: '2025-2026', required: true },
    { field: 'category', label: 'Categorie', type: 'select', options: MERIT_CATEGORY_OPTIONS, defaultValue: 'predare', required: true },
    { field: 'status', label: 'Status', type: 'select', options: MERIT_STATUS_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'score', label: 'Punctaj (0-100)', type: 'number', defaultValue: 0, min: 0, max: 100, step: 0.01 },
    { field: 'committee_name', label: 'Comisie', type: 'text' },
    { field: 'decision_date', label: 'Data deciziei', type: 'date' },
    { field: 'funded', label: 'Finantat', type: 'boolean', defaultValue: false },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  detailChildren: [MERIT_DOCUMENTS_CHILD, MERIT_SCORES_CHILD, MERIT_APPEALS_CHILD, MERIT_FINAL_DECISIONS_CHILD, MERIT_RESULT_ISSUES_CHILD],
  emptyText: 'Nu exista inregistrari pentru gradatii.',
};

const PORTFOLIOS_RESOURCE: EducationResourceConfig = {
  key: 'portfolios',
  label: 'Portofolii CD',
  icon: 'pi pi-folder-open',
  endpoint: '/api/education/portfolios/records',
  createEndpoint: '/api/education/portfolios/records',
  createWizardRoute: '/education/portfolio/wizard',
  detailSummaryKind: 'portfolio-transfer-summary',
  detailSummaryEndpoint: (row) => `/api/education/portfolios/records/${String(row['id'] ?? '')}/transfer-summary`,
  pdfEndpoint: (row) => `/api/education/portfolios/records/${row['id']}/pdf`,
  pdfFilename: (row) => `portofoliu-cd-${row['portfolio_code'] ?? row['id']}.pdf`,
  pdfActionLabel: 'Portofoliu PDF',
  readPermission: 'education.portfolios.read',
  managePermission: 'education.portfolios.manage',
  description: 'Opis, sectiuni, transfer digital, autenticitate, consimtamant si retentie.',
  columns: [
    { field: 'portfolio_code', header: 'Cod', sortable: true, filter: 'text', width: '10rem' },
    { field: 'owner_name', header: 'Titular', sortable: true, filter: 'text', width: '18rem' },
    { field: 'owner_role', header: 'Functie', sortable: true, filter: 'text', width: '12rem' },
    { field: 'school_year', header: 'An', sortable: true, filter: 'text', width: '8rem' },
    { field: 'status', header: 'Status', type: 'tag', sortable: true, filter: 'select', options: PORTFOLIO_STATUS_OPTIONS, width: '10rem' },
    { field: 'section_count', header: 'Sectiuni', type: 'number', sortable: true, width: '8rem' },
    { field: 'transfer_status', header: 'Transfer', type: 'tag', sortable: true, filter: 'select', options: PORTFOLIO_TRANSFER_OPTIONS, width: '10rem' },
    { field: 'retention_until', header: 'Retentie', sortable: true, width: '9rem' },
  ],
  createFields: [
    { field: 'owner_name', label: 'Titular', type: 'text', wide: true, required: true },
    { field: 'owner_role', label: 'Functie', type: 'text', required: true },
    { field: 'school_year', label: 'An scolar', type: 'text', defaultValue: '2025-2026', required: true },
    { field: 'status', label: 'Status', type: 'select', options: PORTFOLIO_STATUS_OPTIONS, defaultValue: 'draft', required: true },
    { field: 'section_count', label: 'Numar sectiuni', type: 'number', defaultValue: 0 },
    { field: 'last_updated_on', label: 'Ultima actualizare', type: 'date' },
    { field: 'retention_until', label: 'Retentie pana la', type: 'date' },
    { field: 'transfer_status', label: 'Status transfer', type: 'select', options: PORTFOLIO_TRANSFER_OPTIONS, defaultValue: 'none', required: true },
    { field: 'authenticity_declared', label: 'Declaratie autenticitate', type: 'boolean', defaultValue: false },
    { field: 'consent_captured', label: 'Consimtamant capturat', type: 'boolean', defaultValue: false },
    { field: 'custodian', label: 'Custode', type: 'text' },
    { field: 'notes', label: 'Note', type: 'textarea', wide: true },
  ],
  emptyText: 'Nu exista portofolii pentru filtrele curente.',
  detailChildren: [PORTFOLIO_DOCUMENTS_CHILD, PORTFOLIO_CHECKLIST_CHILD, PORTFOLIO_OPIS_CHILD, PORTFOLIO_CUSTODY_CHILD, PORTFOLIO_TRANSFERS_CHILD, PORTFOLIO_REVIEWS_CHILD, PORTFOLIO_VALORIFICATIONS_CHILD],
};

const PORTFOLIO_SECTIONS_RESOURCE: EducationResourceConfig = {
  key: 'portfolio_sections',
  label: 'Structura portofoliu',
  icon: 'pi pi-list-check',
  endpoint: '/api/education/portfolios/sections',
  createEndpoint: '',
  allowCreate: false,
  readPermission: 'education.portfolios.read',
  description: 'Structura-cadru: sectiuni, componente, exemple de documente, date sensibile si retentie.',
  columns: [
    { field: 'sort_order', header: 'Ordine', type: 'number', sortable: true, width: '7rem' },
    { field: 'section_code', header: 'Sectiune', sortable: true, filter: 'text', width: '12rem' },
    { field: 'component_code', header: 'Componenta', sortable: true, filter: 'text', width: '14rem' },
    { field: 'label_ro', header: 'Denumire', sortable: true, filter: 'text', width: '22rem' },
    { field: 'required', header: 'Obligatoriu', type: 'boolean', sortable: true, width: '8rem' },
    { field: 'sensitive_data', header: 'Date sensibile', type: 'boolean', sortable: true, width: '9rem' },
    { field: 'retention_rule', header: 'Retentie', sortable: true, filter: 'text', width: '18rem' },
  ],
  createFields: [],
  emptyText: 'Structura portofoliului nu este inca populata.',
};

export const EDUCATION_ANY_READ_PERMISSIONS = [
  'education.read',
  'education.governance.read',
  'education.decisions.read',
  'education.managerial.read',
  'education.regulations.read',
  'education.personnel.read',
  'education.evaluations.read',
  'education.declarations.read',
  'education.mobility.read',
  'education.gradatii.read',
  'education.portfolios.read',
];

export const EDUCATION_ROUTE_TABS: EducationRouteTab[] = [
  { path: 'dashboard', label: 'Dashboard', icon: 'pi pi-chart-bar', permissions: EDUCATION_ANY_READ_PERMISSIONS },
  {
    path: 'governance',
    label: 'Guvernanta',
    icon: 'pi pi-users',
    permissions: ['education.governance.read', 'education.decisions.read', 'education.managerial.read', 'education.regulations.read'],
  },
  {
    path: 'personnel',
    label: 'Personal',
    icon: 'pi pi-id-card',
    permissions: ['education.personnel.read', 'education.evaluations.read', 'education.declarations.read', 'education.mobility.read', 'education.gradatii.read'],
  },
  {
    path: 'portfolio',
    label: 'Portofolii',
    icon: 'pi pi-folder-open',
    permissions: ['education.portfolios.read'],
  },
  {
    path: 'compliance',
    label: 'Conformitate',
    icon: 'pi pi-shield',
    permissions: ['education.read'],
  },
];

export const EDUCATION_DASHBOARD_CARDS = DASHBOARD_CARDS;

export const EDUCATION_GOVERNANCE_SECTION: EducationSectionConfig = {
  key: 'governance',
  label: 'Guvernanta educationala',
  icon: 'pi pi-users',
  description: 'Componenta organismelor, sedinte, hotarari, decizii, dosare manageriale si regulamente institutionale.',
  resources: [
    GOVERNANCE_BODIES_RESOURCE,
    GOVERNANCE_MEMBERSHIPS_RESOURCE,
    MEETINGS_RESOURCE,
    CP_MEETINGS_RESOURCE,
    COMMITTEES_RESOURCE,
    DECISIONS_RESOURCE,
    MANAGERIAL_RESOURCE,
    DIRECTOR_PORTFOLIO_RESOURCE,
    ADJUNCT_DIRECTOR_PORTFOLIO_RESOURCE,
    REGULATIONS_RESOURCE,
  ],
};

export const EDUCATION_PERSONNEL_SECTION: EducationSectionConfig = {
  key: 'personnel',
  label: 'Personal educational',
  icon: 'pi pi-id-card',
  description: 'Cadre didactice, evaluari, declaratii, mobilitate si gradatii.',
  resources: [
    PERSONNEL_RESOURCE,
    EVALUATIONS_RESOURCE,
    DECLARATIONS_RESOURCE,
    MOBILITY_RESOURCE,
    MERIT_RESOURCE,
  ],
};

export const EDUCATION_PORTFOLIO_SECTION: EducationSectionConfig = {
  key: 'portfolio',
  label: 'Portofolii profesionale',
  icon: 'pi pi-folder-open',
  description: 'Portofolii profesionale, transfer, retentie si structura-cadru.',
  resources: [
    PORTFOLIOS_RESOURCE,
    PORTFOLIO_SECTIONS_RESOURCE,
  ],
};

export const EDUCATION_COMPLIANCE_SECTION: EducationSectionConfig = {
  key: 'compliance',
  label: 'Conformitate',
  icon: 'pi pi-shield',
  description: 'Urmarirea cerintelor legale si a gradului de implementare.',
  resources: [REQUIREMENTS_RESOURCE, PUBLICATIONS_RESOURCE],
};

export const EDUCATION_SECTIONS: EducationSectionConfig[] = [
  EDUCATION_GOVERNANCE_SECTION,
  EDUCATION_PERSONNEL_SECTION,
  EDUCATION_PORTFOLIO_SECTION,
  EDUCATION_COMPLIANCE_SECTION,
];
