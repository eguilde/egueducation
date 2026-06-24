import { Page, Route } from '@playwright/test';

export interface MockDirectorCockpitResponse {
  school_year: string;
  institution_id: string;
  governance: {
    total_meetings: number;
    scheduled_meetings: number;
    meetings_without_minute: number;
    meetings_without_vote: number;
    published_resolutions: number;
  };
  portfolios: {
    total_records: number;
    draft_records: number;
    review_records: number;
    returned_records: number;
    validated_records: number;
    transfer_in_progress: number;
  };
  evaluations: {
    total_records: number;
    submitted_records: number;
    reviewed_records: number;
    contested_records: number;
    approved_records: number;
    communicated_documents: number;
  };
  managerial: {
    total_dossiers: number;
    draft_dossiers: number;
    review_dossiers: number;
    approved_dossiers: number;
    published_documents: number;
    workflow_open_steps: number;
  };
  personnel: {
    total_records: number;
    active_records: number;
    portfolio_enabled: number;
    evaluation_pending: number;
    mobility_cases: number;
  };
  compliance: {
    total_requirements: number;
    implemented_requirements: number;
    partial_requirements: number;
    pending_publications: number;
    anonymization_pending: number;
  };
  alerts: Array<{
    id: string;
    title: string;
    summary: string;
    status: string;
    route: string;
    priority: number;
  }>;
  recommended_links: Array<{
    key: string;
    label: string;
    route: string;
  }>;
}

export async function bootstrapAuthenticatedDirector(page: Page): Promise<void> {
  await bootstrapAuthenticatedEducationUser(page, {
    userId: 'director-1',
    name: 'Director Demo',
    email: 'director@example.test',
    roles: ['director'],
    permissions: [
      'registratura.read',
      'registratura.manage',
      'registratura.write',
      'workflow.read',
      'workflow.write',
      'earchiva.read',
      'earchiva.write',
      'education.read',
      'education.governance.read',
      'education.governance.manage',
      'education.managerial.read',
      'education.managerial.manage',
      'education.regulations.read',
      'education.regulations.manage',
      'education.personnel.read',
      'education.personnel.manage',
      'education.personnel.files.read',
      'education.personnel.files.manage',
      'education.personnel.access.read',
      'education.personnel.access.manage',
      'education.evaluations.read',
      'education.evaluations.manage',
      'education.declarations.read',
      'education.declarations.manage',
      'education.mobility.read',
      'education.mobility.manage',
      'education.gradatii.read',
      'education.gradatii.manage',
      'education.portfolios.read',
      'education.portfolios.manage',
    ],
    modules: [
      { code: 'education', active: true },
      { code: 'registratura', active: true },
      { code: 'workflow', active: true },
      { code: 'earchiva', active: true },
      { code: 'admin', active: true },
    ],
  });
}

export async function bootstrapAuthenticatedSecretariat(page: Page): Promise<void> {
  await bootstrapAuthenticatedEducationUser(page, {
    userId: 'secretar-1',
    name: 'Secretariat Demo',
    email: 'secretariat@example.test',
    roles: ['secretar'],
    permissions: [
      'education.read',
      'education.governance.read',
      'education.personnel.read',
      'education.portfolios.read',
      'education.managerial.read',
    ],
    modules: [{ code: 'education', active: true }],
  });
}

export async function bootstrapAuthenticatedEducationUser(
  page: Page,
  user: {
    userId: string;
    name: string;
    email: string;
    roles: string[];
    permissions?: string[];
    modules?: Array<{ code: string; active: boolean }>;
    institutionId?: string;
    institutionName?: string;
  },
): Promise<void> {
  const expiresAt = Math.floor(Date.now() / 1000) + 3600;
  const payload = Buffer.from(
    JSON.stringify({
      sub: user.userId,
      name: user.name,
      email: user.email,
      roles: user.roles,
    }),
  ).toString('base64url');

  await page.addInitScript(({ encodedPayload }) => {
    localStorage.setItem('egueducation_auth_has_session', '1');
    localStorage.setItem('egueducation_auth_access_token', 'test-access-token');
    localStorage.setItem('egueducation_auth_expires_at', String(Math.floor(Date.now() / 1000) + 3600));
    localStorage.setItem('egueducation_auth_id_token', `header.${encodedPayload}.signature`);
  }, { encodedPayload: payload });

  await page.addInitScript(({ session }) => {
    localStorage.setItem('egueducation_e2e_session', JSON.stringify(session));
  }, {
    session: {
      profile: {
        sub: user.userId,
        email: user.email,
        name: user.name,
        roles: user.roles,
      },
      accessToken: 'test-access-token',
      expiresAt,
      session: {
        user: {
          id: user.userId,
          sub: user.userId,
          name: user.name,
          email: user.email,
          locale: 'ro',
          roles: user.roles,
        },
        institution_id: user.institutionId ?? 'school-1',
        institution_name: user.institutionName ?? 'Scoala Gimnaziala Demonstrativa',
        permissions: user.permissions ?? [],
        modules: user.modules ?? [],
        authentication: ['oidc'],
        gdpr_capabilities: [],
      },
    },
  });
}

export async function registerEducationApiMocks(
  page: Page,
  overrides?: {
    cockpit?: MockDirectorCockpitResponse;
    governanceFinalizationSummary?: Record<string, unknown>;
    governanceMeetingDetail?: Record<string, unknown>;
    governanceMinuteItems?: Array<Record<string, unknown>>;
    governanceResolutionItems?: Array<Record<string, unknown>>;
    me?: Partial<MockSessionResponse>;
    onCreateMeeting?: (payload: Record<string, unknown>) => void;
    onPatchMeeting?: (payload: Record<string, unknown>) => void;
    onCreateMinute?: (meetingId: string, payload: Record<string, unknown>) => void;
    onCreateVote?: (meetingId: string, payload: Record<string, unknown>) => void;
    onCreateResolution?: (meetingId: string, payload: Record<string, unknown>) => void;
    onCreateManagerial?: (payload: Record<string, unknown>) => void;
    onCreatePortfolio?: (payload: Record<string, unknown>) => void;
    onCreatePersonnel?: (payload: Record<string, unknown>) => void;
    onCreateEvaluation?: (payload: Record<string, unknown>) => void;
    onCreateDeclaration?: (payload: Record<string, unknown>) => void;
    onCreateMobility?: (payload: Record<string, unknown>) => void;
    onCreateMerit?: (payload: Record<string, unknown>) => void;
  },
): Promise<void> {
  const cockpit = overrides?.cockpit ?? defaultCockpit();
  let portfolioTransferState: 'pregatit' | 'trimis' | 'receptionat' | 'inchis' = 'pregatit';
  let portfolioSelfServiceStatus: 'draft' | 'submitted' | 'validated' | 'transferred' | 'archived' = 'draft';
  const portfolioValorifications = [
    {
      id: 'valorification-1',
      valorification_code: 'VAL-2026-0001',
      scope: 'evaluare_profesionala',
      status: 'finalizat',
      requested_by: 'Director Demo',
      target_institution: 'Scoala Gimnaziala Demonstrativa',
      target_reference: 'EVA-2026-0001',
      started_on: '2026-05-20',
      completed_on: '2026-06-10',
      notes: 'Portofoliul a sustinut evaluarea profesionala anuala.',
    },
    {
      id: 'valorification-2',
      valorification_code: 'VAL-2026-0002',
      scope: 'mobilitate',
      status: 'transmis',
      requested_by: 'Secretariat Demo',
      target_institution: 'Scoala Gimnaziala Partenera',
      target_reference: 'MOB-2026-0001',
      started_on: '2026-06-24',
      completed_on: '',
      notes: 'Extras procedural transmis pentru mobilitate.',
    },
    {
      id: 'valorification-3',
      valorification_code: 'VAL-2026-0003',
      scope: 'gradatie_merit',
      status: 'validat',
      requested_by: 'Comisia de evaluare',
      target_institution: 'Comisia judeteana',
      target_reference: 'MERIT-2026-0001',
      started_on: '2026-06-12',
      completed_on: '2026-06-22',
      notes: 'Dovezile pentru gradatie au fost validate.',
    },
  ];

  await page.route('**/api/**', async (route) => {
    const url = route.request().url();
    const method = route.request().method();

    if (url.includes('/api/config')) {
      return fulfillJson(route, {
        institutionId: 'school-1',
        institutionName: 'Scoala Gimnaziala Demonstrativa',
        customer: { name: 'EguEducation Demo' },
        service: { title: 'EguEducation' },
      });
    }

    if (url.includes('/api/oidc/.well-known/')) {
      return fulfillJson(route, {
        issuer: 'http://127.0.0.1:4200/api/oidc',
        authorization_endpoint: 'http://127.0.0.1:4200/api/oidc/authorize',
        token_endpoint: 'http://127.0.0.1:4200/api/oidc/token',
        userinfo_endpoint: 'http://127.0.0.1:4200/api/oidc/userinfo',
        jwks_uri: 'http://127.0.0.1:4200/api/oidc/jwks',
        revocation_endpoint: 'http://127.0.0.1:4200/api/oidc/revoke',
        end_session_endpoint: 'http://127.0.0.1:4200/api/oidc/logout',
        response_types_supported: ['code'],
        grant_types_supported: ['authorization_code', 'refresh_token'],
        subject_types_supported: ['public'],
        id_token_signing_alg_values_supported: ['RS256'],
        token_endpoint_auth_methods_supported: ['none'],
        code_challenge_methods_supported: ['S256'],
        scopes_supported: ['openid', 'profile', 'email', 'phone', 'offline_access'],
        claims_supported: ['sub', 'email', 'name', 'phone_number', 'roles'],
      });
    }

    if (url.endsWith('/api/me')) {
      return fulfillJson(route, buildSessionResponse(overrides?.me));
    }

    if (url.endsWith('/api/auth/role-catalog')) {
      return fulfillJson(route, {
        roles: [
          {
            code: 'director',
            label: 'Director',
            description: 'Director unitate de invatamant',
            permissions: ['education.governance.manage'],
            positions: ['director'],
          },
          {
            code: 'secretar',
            label: 'Secretariat',
            description: 'Rol operational pentru gestionarea fluxurilor educationale',
            permissions: ['education.governance.read', 'education.personnel.read'],
            positions: ['secretar'],
          },
          {
            code: 'registrator',
            label: 'Registratura',
            description: 'Rol operational pentru intrari si evidenta documentelor',
            permissions: ['education.read'],
            positions: ['registrator'],
          },
          {
            code: 'profesor',
            label: 'Profesor',
            description: 'Rol operational pentru fluxurile personale ale cadrului didactic',
            permissions: ['education.portfolios.read', 'education.evaluations.read', 'education.declarations.read'],
            positions: ['profesor'],
          },
        ],
      });
    }

    if (url.endsWith('/api/auth/role-positions')) {
      return fulfillJson(route, {
        items: [
          {
            position_code: 'director',
            position_name: 'Director',
            role_code: 'director',
            role_label: 'Director',
          },
          {
            position_code: 'secretar',
            position_name: 'Secretar',
            role_code: 'secretar',
            role_label: 'Secretariat',
          },
          {
            position_code: 'registrator',
            position_name: 'Registrator',
            role_code: 'registrator',
            role_label: 'Registratura',
          },
          {
            position_code: 'profesor',
            position_name: 'Profesor',
            role_code: 'profesor',
            role_label: 'Profesor',
          },
        ],
      });
    }

    if (url.endsWith('/api/education/director/cockpit')) {
      return fulfillJson(route, cockpit);
    }

    if (url.includes('/api/education/governance/meetings') && method === 'POST') {
      const body = route.request().postDataJSON() as Record<string, unknown>;
      const resolutionMatch = url.match(/\/api\/education\/governance\/meetings\/([^/]+)\/resolutions$/);
      if (resolutionMatch) {
        overrides?.onCreateResolution?.(resolutionMatch[1], body);
        return fulfillJson(route, {
          id: 'resolution-new-1',
          meeting_id: resolutionMatch[1],
          resolution_code: 'HCA-2026-001',
          ...body,
        });
      }

      const voteMatch = url.match(/\/api\/education\/governance\/meetings\/([^/]+)\/votes$/);
      if (voteMatch) {
        overrides?.onCreateVote?.(voteMatch[1], body);
        return fulfillJson(route, {
          id: 'vote-new-1',
          meeting_id: voteMatch[1],
          ...body,
        });
      }

      const match = url.match(/\/api\/education\/governance\/meetings\/([^/]+)\/minutes$/);
      if (match) {
        overrides?.onCreateMinute?.(match[1], body);
        return fulfillJson(route, {
          id: 'minute-new-1',
          meeting_id: match[1],
          ...body,
        });
      }

      overrides?.onCreateMeeting?.(body);
      return fulfillJson(route, {
        id: 'meeting-new-1',
        ...body,
      });
    }

    if (url.endsWith('/api/education/managerial/records') && method === 'POST') {
      const body = route.request().postDataJSON() as Record<string, unknown>;
      overrides?.onCreateManagerial?.(body);
      return fulfillJson(route, {
        id: 'managerial-new-1',
        dossier_code: 'MNG-2026-001',
        ...body,
      });
    }

    if (/\/api\/education\/managerial\/records(?:\?.*)?$/.test(url) && method === 'GET') {
      if (url.includes('filter.dossier_type=adjunct_director_portfolio')) {
        return fulfillJson(route, {
          items: [
            {
              id: 'managerial-adjunct-1',
              dossier_code: 'MGR-ADJ-2026-001',
              school_year: '2025-2026',
              dossier_type: 'adjunct_director_portfolio',
              title: 'Portofoliul directorului adjunct al unitatii de invatamant',
              status: 'draft',
              owner_name: 'Ioana Marin',
              due_on: '2026-07-20',
              publication_required: true,
              institution_id: 'school-1',
              summary: 'Registru managerial explicit pentru portofoliul directorului adjunct.',
            },
          ],
          total: 1,
          page: 1,
          pageSize: 10,
        });
      }

      if (url.includes('filter.dossier_type=director_portfolio')) {
        return fulfillJson(route, {
          items: [
            {
              id: 'managerial-portfolio-1',
              dossier_code: 'MGR-DIR-2026-001',
              school_year: '2025-2026',
              dossier_type: 'director_portfolio',
              title: 'Portofoliul directorului unitatii de invatamant',
              status: 'in_review',
              owner_name: 'Raluca Stan',
              due_on: '2026-07-10',
              publication_required: true,
              institution_id: 'school-1',
              summary: 'Registru managerial explicit pentru portofoliul directorului.',
            },
          ],
          total: 1,
          page: 1,
          pageSize: 10,
        });
      }

      return fulfillJson(route, {
        items: [
          {
            id: 'managerial-portfolio-1',
            dossier_code: 'MGR-DIR-2026-001',
            school_year: '2025-2026',
            dossier_type: 'director_portfolio',
            title: 'Portofoliul directorului unitatii de invatamant',
            status: 'in_review',
            owner_name: 'Raluca Stan',
            due_on: '2026-07-10',
            publication_required: true,
            institution_id: 'school-1',
            summary: 'Registru managerial explicit pentru portofoliul directorului.',
          },
          {
            id: 'managerial-adjunct-1',
            dossier_code: 'MGR-ADJ-2026-001',
            school_year: '2025-2026',
            dossier_type: 'adjunct_director_portfolio',
            title: 'Portofoliul directorului adjunct al unitatii de invatamant',
            status: 'draft',
            owner_name: 'Ioana Marin',
            due_on: '2026-07-20',
            publication_required: true,
            institution_id: 'school-1',
            summary: 'Registru managerial explicit pentru portofoliul directorului adjunct.',
          },
        ],
        total: 2,
        page: 1,
        pageSize: 10,
      });
    }

    if (/\/api\/education\/managerial\/records\/managerial-portfolio-1(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        id: 'managerial-portfolio-1',
        dossier_code: 'MGR-DIR-2026-001',
        school_year: '2025-2026',
        dossier_type: 'director_portfolio',
        title: 'Portofoliul directorului unitatii de invatamant',
        status: 'in_review',
        owner_name: 'Raluca Stan',
        due_on: '2026-07-10',
        publication_required: true,
        institution_id: 'school-1',
        summary: 'Registru managerial explicit pentru portofoliul directorului.',
      });
    }

    if (/\/api\/education\/managerial\/records\/managerial-adjunct-1(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        id: 'managerial-adjunct-1',
        dossier_code: 'MGR-ADJ-2026-001',
        school_year: '2025-2026',
        dossier_type: 'adjunct_director_portfolio',
        title: 'Portofoliul directorului adjunct al unitatii de invatamant',
        status: 'draft',
        owner_name: 'Ioana Marin',
        due_on: '2026-07-20',
        publication_required: true,
        institution_id: 'school-1',
        summary: 'Registru managerial explicit pentru portofoliul directorului adjunct.',
      });
    }

    if (/\/api\/education\/managerial\/records\/managerial-portfolio-1\/portfolio-summary(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        dossier: {
          id: 'managerial-portfolio-1',
          dossier_code: 'MGR-DIR-2026-001',
          dossier_type: 'director_portfolio',
          title: 'Portofoliul directorului unitatii de invatamant',
          school_year: '2025-2026',
          status: 'in_review',
          owner_name: 'Raluca Stan',
          publication_required: true,
        },
        portfolio: {
          matched_personnel: 1,
          managerial_documents: 2,
          mandatory_documents: 2,
          approved_documents: 1,
          published_documents: 0,
          publication_required_documents: 1,
          missing_mandatory_categories: [],
        },
        workflow: {
          total_steps: 2,
          completed_steps: 1,
          open_steps: 1,
          signature_steps: 1,
          completed_signature_steps: 0,
        },
        personnel_file: {
          matched_documents: 1,
          management_documents: 1,
          sensitive_documents: 1,
          mirrored_references: 0,
        },
        readiness: {
          ready_for_review: true,
          ready_for_publication: false,
          blockers: ['mandatory_documents_pending'],
        },
      });
    }

    if (/\/api\/education\/managerial\/records\/managerial-adjunct-1\/portfolio-summary(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        dossier: {
          id: 'managerial-adjunct-1',
          dossier_code: 'MGR-ADJ-2026-001',
          dossier_type: 'adjunct_director_portfolio',
          title: 'Portofoliul directorului adjunct al unitatii de invatamant',
          school_year: '2025-2026',
          status: 'draft',
          owner_name: 'Ioana Marin',
          publication_required: true,
        },
        portfolio: {
          matched_personnel: 1,
          managerial_documents: 1,
          mandatory_documents: 1,
          approved_documents: 0,
          published_documents: 0,
          publication_required_documents: 1,
          missing_mandatory_categories: ['evaluare'],
        },
        workflow: {
          total_steps: 2,
          completed_steps: 0,
          open_steps: 2,
          signature_steps: 1,
          completed_signature_steps: 0,
        },
        personnel_file: {
          matched_documents: 1,
          management_documents: 1,
          sensitive_documents: 1,
          mirrored_references: 0,
        },
        readiness: {
          ready_for_review: false,
          ready_for_publication: false,
          blockers: ['mandatory_documents_pending', 'workflow_open_steps'],
        },
      });
    }

    if (/\/api\/education\/managerial\/records\/managerial-portfolio-1\/documents(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        items: [
          {
            id: 'mdoc-1',
            dossier_id: 'managerial-portfolio-1',
            document_code: 'MDOC-DIR-2026-0001',
            document_category: 'evidenta',
            title: 'Opis documente portofoliu director',
            document_status: 'approved',
            version_label: 'v1.0',
            mandatory: true,
            publication_required: false,
            registered_on: '2026-06-20',
            approved_on: '2026-06-21',
            owner_name: 'Raluca Stan',
            file_reference: 'REG-MGR-DIR-2026-001',
            institution_id: 'school-1',
            notes: 'Opis si index procedural.',
          },
          {
            id: 'mdoc-2',
            dossier_id: 'managerial-portfolio-1',
            document_code: 'MDOC-DIR-2026-0002',
            document_category: 'planificare',
            title: 'Plan managerial si documente de baza ale portofoliului directorului',
            document_status: 'in_review',
            version_label: 'v0.9',
            mandatory: true,
            publication_required: true,
            registered_on: '2026-06-22',
            approved_on: '',
            owner_name: 'Raluca Stan',
            file_reference: 'REG-MGR-DIR-2026-002',
            institution_id: 'school-1',
            notes: 'Pachetul de baza este in curs de avizare.',
          },
        ],
        total: 2,
        page: 1,
        pageSize: 10,
      });
    }

    if (/\/api\/education\/managerial\/records\/managerial-portfolio-1\/workflow(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        items: [
          {
            id: 'mflow-1',
            dossier_id: 'managerial-portfolio-1',
            stage_order: 1,
            stage_type: 'elaborare',
            status: 'completed',
            assigned_to: 'Raluca Stan',
            due_on: '2026-06-20',
            completed_on: '2026-06-20',
            requires_signature: false,
            decision_reference: '',
            institution_id: 'school-1',
            outcome_note: 'Structura initiala a fost pregatita.',
          },
          {
            id: 'mflow-2',
            dossier_id: 'managerial-portfolio-1',
            stage_order: 2,
            stage_type: 'avizare_cp',
            status: 'in_progress',
            assigned_to: 'Secretar CP',
            due_on: '2026-07-02',
            completed_on: '',
            requires_signature: true,
            decision_reference: 'PV-CP-2026-PORT-01',
            institution_id: 'school-1',
            outcome_note: 'Se asteapta avizarea in CP.',
          },
        ],
        total: 2,
        page: 1,
        pageSize: 10,
      });
    }

    if (url.endsWith('/api/education/portfolios/records') && method === 'POST') {
      const body = route.request().postDataJSON() as Record<string, unknown>;
      overrides?.onCreatePortfolio?.(body);
      return fulfillJson(route, {
        id: 'portfolio-new-1',
        portfolio_code: 'PORT-2026-001',
        ...body,
      });
    }

    if (url.endsWith('/api/education/personnel/records') && method === 'POST') {
      const body = route.request().postDataJSON() as Record<string, unknown>;
      overrides?.onCreatePersonnel?.(body);
      return fulfillJson(route, {
        id: 'personnel-new-1',
        employee_code: 'EMP-NEW-1',
        ...body,
      });
    }

    if (url.endsWith('/api/education/evaluations/records') && method === 'POST') {
      const body = route.request().postDataJSON() as Record<string, unknown>;
      overrides?.onCreateEvaluation?.(body);
      return fulfillJson(route, {
        id: 'evaluation-new-1',
        evaluation_code: 'EVAL-2026-001',
        ...body,
      });
    }

    if (url.endsWith('/api/education/declarations/records') && method === 'POST') {
      const body = route.request().postDataJSON() as Record<string, unknown>;
      overrides?.onCreateDeclaration?.(body);
      return fulfillJson(route, {
        id: 'declaration-new-1',
        declaration_code: 'DECL-2026-001',
        ...body,
      });
    }

    if (url.endsWith('/api/education/mobility/records') && method === 'POST') {
      const body = route.request().postDataJSON() as Record<string, unknown>;
      overrides?.onCreateMobility?.(body);
      return fulfillJson(route, {
        id: 'mobility-new-1',
        case_code: 'MOB-2026-001',
        ...body,
      });
    }

    if (url.endsWith('/api/education/gradatii/records') && method === 'POST') {
      const body = route.request().postDataJSON() as Record<string, unknown>;
      overrides?.onCreateMerit?.(body);
      return fulfillJson(route, {
        id: 'merit-new-1',
        grant_code: 'MERIT-2026-001',
        ...body,
      });
    }

    if (url.includes('/api/education/governance/meetings/filters')) {
      return fulfillJson(route, {
        school_years: ['2025-2026'],
        organisms: ['ca', 'cp', 'ceac'],
        meeting_types: ['ordinary', 'extraordinary'],
        statuses: ['draft', 'scheduled', 'held', 'published'],
      });
    }

    if (/\/api\/education\/governance\/meetings\?[^#]*filter\.organism=cp/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        items: [
          {
            id: 'meeting-2',
            school_year: '2025-2026',
            organism: 'cp',
            title: 'Sedinta CP pentru analiza rezultatelor',
            status: 'draft',
            meeting_date: '2026-07-02',
            participants_count: 16,
          },
        ],
        total: 1,
        page: 1,
        pageSize: url.includes('pageSize=100') ? 100 : 20,
      });
    }

    if (/\/api\/education\/governance\/meetings\/meeting-1(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        id: 'meeting-1',
        school_year: '2025-2026',
        organism: 'ca',
        title: 'Sedinta CA pentru aprobarea planului managerial',
        meeting_type: 'ordinary',
        status: 'scheduled',
        quorum_required: 5,
        participants_count: 7,
        meeting_date: '2026-06-30',
        location: 'Sala profesorală',
        chairperson: 'Prof. Director Demo',
        secretary_name: 'Secretar Demo',
        institution_id: 'school-1',
        summary: 'Sedinta de coordonare manageriala.',
        ...(overrides?.governanceMeetingDetail ?? {}),
      });
    }

    if (/\/api\/education\/governance\/meetings\/meeting-2(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        id: 'meeting-2',
        school_year: '2025-2026',
        organism: 'cp',
        title: 'Sedinta CP pentru analiza rezultatelor',
        meeting_type: 'ordinary',
        status: 'draft',
        quorum_required: 10,
        participants_count: 16,
        meeting_date: '2026-07-02',
        location: 'Sala profesorală',
        chairperson: 'Prof. Director Demo',
        secretary_name: 'Secretar Demo',
        institution_id: 'school-1',
        summary: 'Sedinta profesorală dedicata analizei rezultatelor si masurilor educationale.',
      });
    }

    if (/\/api\/education\/governance\/meetings\/meeting-1(?:\?.*)?$/.test(url) && method === 'PATCH') {
      const body = route.request().postDataJSON() as Record<string, unknown>;
      overrides?.onPatchMeeting?.(body);
      return fulfillJson(route, {
        id: 'meeting-1',
        school_year: '2025-2026',
        organism: 'ca',
        title: 'Sedinta CA pentru aprobarea planului managerial',
        meeting_type: 'ordinary',
        status: String(body['status'] ?? 'scheduled'),
        quorum_required: Number(body['quorum_required'] ?? 5),
        participants_count: Number(body['participants_count'] ?? 7),
        meeting_date: '2026-06-30',
        location: 'Sala profesorală',
        chairperson: 'Prof. Director Demo',
        secretary_name: 'Secretar Demo',
        institution_id: 'school-1',
        summary: 'Sedinta de coordonare manageriala.',
      });
    }

    if (url.includes('/api/education/governance/meetings/meeting-1/finalization-summary') && method === 'GET') {
      return fulfillJson(route, {
        meeting: {
          id: 'meeting-1',
          school_year: '2025-2026',
          organism: 'ca',
          title: 'Sedinta CA pentru aprobarea planului managerial',
          meeting_type: 'ordinary',
          status: 'scheduled',
          quorum_required: 5,
          participants_count: 7,
          meeting_date: '2026-06-30',
          location: 'Sala profesorală',
          chairperson: 'Prof. Director Demo',
          secretary_name: 'Secretar Demo',
          institution_id: 'school-1',
          summary: 'Sedinta de coordonare manageriala.',
        },
        participants: {
          recorded_participants: 7,
          present_participants: 4,
          signed_participants: 4,
          voting_participants: 6,
        },
        votes: {
          total: 1,
          adopted: 1,
          requires_follow_up: 1,
          missing_resolutions: 1,
        },
        minutes: {
          total: 0,
          requires_publication: 0,
          open_follow_up_items: 0,
        },
        resolutions: {
          total: 0,
          ready_for_publication: 0,
          published: 0,
          pending_publication: 0,
          pending_anonymization: 0,
        },
        documents: {
          total: 1,
          process_verbal_documents: 0,
          published_process_verbals: 0,
        },
        readiness: {
          ready_to_close: false,
          ready_to_publish: false,
          blockers: ['quorum_incomplete', 'minutes_missing', 'resolutions_missing_for_adopted_votes'],
        },
        ...(overrides?.governanceFinalizationSummary ?? {}),
      });
    }

    if (url.includes('/api/education/governance/meetings/meeting-1/minutes/') && url.endsWith('/pdf') && method === 'GET') {
      return route.fulfill({
        status: 200,
        contentType: 'application/pdf',
        headers: {
          'Content-Disposition': 'attachment; filename="proces-verbal-meeting-1-pct-1.pdf"',
        },
        body: 'mock-governance-minute-pdf',
      });
    }

    if (url.includes('/api/education/governance/meetings/meeting-1/resolutions/') && url.endsWith('/pdf') && method === 'GET') {
      return route.fulfill({
        status: 200,
        contentType: 'application/pdf',
        headers: {
          'Content-Disposition': 'attachment; filename="hotarare-HCA-2026-001.pdf"',
        },
        body: 'mock-governance-resolution-pdf',
      });
    }

    if (
      url.includes('/api/education/governance/meetings') &&
      method === 'GET' &&
      !url.includes('/minutes') &&
      !url.includes('/votes') &&
      !url.includes('/resolutions') &&
      !url.includes('/filters') &&
      !/\/api\/education\/governance\/meetings\/[^/?]+$/.test(url)
    ) {
      return fulfillJson(route, {
        items: [
          {
            id: 'meeting-1',
            school_year: '2025-2026',
            organism: 'ca',
            title: 'Sedinta CA pentru aprobarea planului managerial',
            status: 'scheduled',
            meeting_date: '2026-06-30',
            participants_count: 7,
          },
          {
            id: 'meeting-2',
            school_year: '2025-2026',
            organism: 'cp',
            title: 'Sedinta CP pentru analiza rezultatelor',
            status: 'draft',
            meeting_date: '2026-07-02',
            participants_count: 16,
          },
        ],
        total: 2,
        page: 1,
        pageSize: url.includes('pageSize=100') ? 100 : 20,
      });
    }

    if (url.includes('/api/education/governance/meetings/meeting-1/votes') && method === 'GET') {
      return fulfillJson(route, {
        items: [
          {
            id: 'vote-1',
            meeting_id: 'meeting-1',
            subject_title: 'Aprobarea procedurii de monitorizare',
            agenda_order: 1,
            outcome: 'adoptat',
          },
        ],
        total: 1,
        page: 1,
        pageSize: url.includes('pageSize=100') ? 100 : 10,
      });
    }

    if (url.includes('/api/education/governance/meetings/meeting-1/minutes') && method === 'GET') {
      return fulfillJson(route, {
        items: overrides?.governanceMinuteItems ?? [],
        total: overrides?.governanceMinuteItems?.length ?? 0,
        page: 1,
        pageSize: 10,
      });
    }

    if (url.includes('/api/education/governance/meetings/meeting-1/resolutions') && method === 'GET') {
      return fulfillJson(route, {
        items: overrides?.governanceResolutionItems ?? [],
        total: overrides?.governanceResolutionItems?.length ?? 0,
        page: 1,
        pageSize: 10,
      });
    }

    if (/\/api\/education\/governance\/bodies\/2025-2026__ca\/completeness-summary(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        body: {
          id: '2025-2026__ca',
          school_year: '2025-2026',
          organism: 'ca',
          active_members: 5,
          voting_members: 5,
          chairperson_covered: true,
          secretary_covered: true,
          expired_mandates: 0,
          total_meetings: 2,
          scheduled_meetings: 1,
          held_meetings: 1,
          published_meetings: 0,
          latest_meeting_on: '2026-06-30',
          readiness_status: 'complet',
        },
        membership: {
          active_members: 5,
          voting_members: 5,
          chairperson_covered: true,
          secretary_covered: true,
          expired_mandates: 0,
          member_names: ['Director Demo', 'Secretariat Demo', 'Prof. Elena Ionescu', 'Reprezentant parinti', 'Reprezentant autoritate locala'],
        },
        meetings: {
          total_meetings: 2,
          scheduled_meetings: 1,
          held_meetings: 1,
          published_meetings: 0,
          last_meeting_title: 'Sedinta CA pentru aprobarea planului managerial',
          last_meeting_on: '2026-06-30',
        },
        readiness: {
          ready_for_operation: true,
          ready_for_meetings: true,
          blockers: [],
        },
      });
    }

    if (/\/api\/education\/governance\/bodies\/2025-2026__ca(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        id: '2025-2026__ca',
        school_year: '2025-2026',
        organism: 'ca',
        active_members: 5,
        voting_members: 5,
        chairperson_covered: true,
        secretary_covered: true,
        expired_mandates: 0,
        total_meetings: 2,
        scheduled_meetings: 1,
        held_meetings: 1,
        published_meetings: 0,
        latest_meeting_on: '2026-06-30',
        readiness_status: 'complet',
        institution_id: 'school-1',
      });
    }

    if (/\/api\/education\/personnel\/records\/personnel-1\/portfolio-dossier-summary(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        personnel: {
          id: 'personnel-1',
          full_name: 'Prof. Elena Ionescu',
          role_title: 'Profesor limba romana',
          school_year: '2025-2026',
          has_portfolio: true,
        },
        dossier: {
          total_documents: 6,
          personal_file_documents: 6,
          director_file_documents: 0,
          adjunct_director_file_documents: 0,
          sensitive_documents: 2,
          documents_marked_for_portfolio: 2,
          evaluation_documents: 1,
          administrative_career_documents: 3,
        },
        portfolio: {
          matched_records: 1,
          validated_records: 1,
          total_documents: 5,
          portfolio_scope_documents: 3,
          personnel_scope_documents: 2,
          verified_documents: 4,
          last_updated_on: '2026-06-24',
        },
        relation: {
          mirrored_file_references: 2,
          evaluation_results_enter_personnel_file: true,
          administrative_docs_enter_personnel_file: true,
          institution_may_duplicate_or_separate: true,
          duplication_mode: 'selectiva',
          rules: [
            'Documentele administrative rezultate din evaluare se arhiveaza in dosarul personal.',
            'Documentele administrative privind studii, cariera si formare continua raman in dosarul personal.',
          ],
        },
        readiness: {
          clear_delimitation: true,
          blockers: [],
        },
      });
    }

    if (/\/api\/education\/regulations\/records\/regulation-1\/procedural-summary(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        regulation: {
          id: 'regulation-1',
          regulation_code: 'REGL-2026-001',
          regulation_type: 'roi',
          title: 'Regulament de ordine interioara',
          school_year: '2025-2026',
          status: 'consultation',
          approval_status: 'consulted',
          review_due_on: '2026-09-01',
          approved_on: '',
        },
        versions: {
          total_versions: 2,
          consultation_versions: 1,
          endorsed_versions: 1,
          approved_versions: 0,
          published_versions: 0,
          latest_version: {
            id: 'reg-version-1',
            version_label: 'v0.8',
            version_status: 'consultation',
            approved_on: '',
            effective_from: '2026-09-01',
            published_on: '',
            prepared_by: 'Thomas Galambos',
            file_reference: 'REG-ROI-2026-0008',
          },
        },
        workflow: {
          total_phases: 2,
          completed_phases: 1,
          open_phases: 1,
          returned_phases: 0,
          cancelled_phases: 0,
          feedback_count: 14,
          current_phase: {
            id: 'reg-phase-2',
            phase_order: 2,
            phase_type: 'consultare_publica',
            status: 'active',
            audience: 'Personal, parinti, elevi majori',
            started_on: '2026-06-03',
            due_on: '2026-06-18',
            completed_on: '',
            feedback_count: 14,
            decision_reference: 'ANUNT-ROI-2026-03',
          },
        },
        readiness: {
          ready_for_cp_endorsement: false,
          ready_for_ca_approval: true,
          ready_for_publication: false,
          ready_for_review: true,
          blockers: ['ca_approval_missing'],
        },
      });
    }

    if (/\/api\/education\/regulations\/records\/regulation-1\/versions(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        items: [
          {
            id: 'reg-version-1',
            regulation_id: 'regulation-1',
            version_label: 'v0.8',
            version_status: 'consultation',
            change_summary: 'Forma pusa in consultare publica.',
            approved_on: '',
            effective_from: '2026-09-01',
            published_on: '',
            prepared_by: 'Thomas Galambos',
            file_reference: 'REG-ROI-2026-0008',
            institution_id: 'school-1',
            notes: 'Consultare publica.',
          },
          {
            id: 'reg-version-2',
            regulation_id: 'regulation-1',
            version_label: 'v0.7',
            version_status: 'endorsed',
            change_summary: 'Forma avizata in CP.',
            approved_on: '2026-05-28',
            effective_from: '2026-09-01',
            published_on: '',
            prepared_by: 'Secretar CA',
            file_reference: 'REG-ROI-2026-0006',
            institution_id: 'school-1',
            notes: 'Pregatita pentru aprobare in CA.',
          },
        ],
        total: 2,
        page: 1,
        pageSize: 10,
      });
    }

    if (/\/api\/education\/regulations\/records\/regulation-1\/workflow(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        items: [
          {
            id: 'reg-phase-1',
            regulation_id: 'regulation-1',
            phase_order: 1,
            phase_type: 'redactare',
            status: 'completed',
            audience: 'Grup de lucru ROI',
            started_on: '2026-05-01',
            due_on: '2026-05-20',
            completed_on: '2026-05-18',
            feedback_count: 0,
            decision_reference: '',
            institution_id: 'school-1',
            notes: 'Text finalizat pentru consultare.',
          },
          {
            id: 'reg-phase-2',
            regulation_id: 'regulation-1',
            phase_order: 2,
            phase_type: 'consultare_publica',
            status: 'active',
            audience: 'Personal, parinti, elevi majori',
            started_on: '2026-06-03',
            due_on: '2026-06-18',
            completed_on: '',
            feedback_count: 14,
            decision_reference: 'ANUNT-ROI-2026-03',
            institution_id: 'school-1',
            notes: 'Observatiile se centralizeaza.',
          },
        ],
        total: 2,
        page: 1,
        pageSize: 10,
      });
    }

    if (/\/api\/education\/regulations\/records\/regulation-1(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        id: 'regulation-1',
        regulation_code: 'REGL-2026-001',
        school_year: '2025-2026',
        regulation_type: 'roi',
        title: 'Regulament de ordine interioara',
        status: 'consultation',
        approval_status: 'consulted',
        owner_name: 'Director Demo',
        review_due_on: '2026-09-01',
        approved_on: '',
        institution_id: 'school-1',
        summary: 'Regulament in consultare publica.',
      });
    }

    if (/\/api\/education\/personnel\/records\/personnel-1(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        id: 'personnel-1',
        employee_code: 'PER-2026-0001',
        full_name: 'Prof. Elena Ionescu',
        role_title: 'Profesor limba romana',
        employment_type: 'titular',
        status: 'active',
        evaluation_status: 'finalized',
        mobility_stage: 'none',
        school_year: '2025-2026',
        assigned_unit: 'Corpul A',
        phone: '0712345678',
        email: 'elena.ionescu@example.test',
        has_portfolio: true,
        institution_id: 'school-1',
        notes: 'Cadru didactic demonstrativ.',
      });
    }

    if (url.includes('/api/education/personnel/records/personnel-1/file-documents') && method === 'GET') {
      return fulfillJson(route, {
        items: [
          {
            id: 'file-doc-1',
            document_code: 'PFD-2026-00001',
            document_category: 'evaluare',
            document_title: 'Comunicare rezultat evaluare',
            file_scope: 'dosar_personal',
            confidentiality_level: 'confidential',
            issued_on: '2026-06-20',
            sensitive_data: true,
            included_in_portfolio: false,
          },
          {
            id: 'file-doc-2',
            document_code: 'PFD-2026-00002',
            document_category: 'cariera',
            document_title: 'Adeverinta formare continua',
            file_scope: 'dosar_personal',
            confidentiality_level: 'intern',
            issued_on: '2026-05-12',
            sensitive_data: false,
            included_in_portfolio: true,
          },
        ],
        total: 2,
        page: 1,
        pageSize: 10,
      });
    }

    if (url.includes('/api/education/personnel/records/personnel-1/access-events') && method === 'GET') {
      return fulfillJson(route, {
        items: [
          {
            id: 'access-1',
            event_type: 'consultare',
            actor_name: 'Secretariat Demo',
            actor_role: 'secretar',
            access_channel: 'digital',
            accessed_on: '2026-06-24',
            closed_on: '',
            sensitive_scope: true,
          },
        ],
        total: 1,
        page: 1,
        pageSize: 10,
      });
    }

    if (url.includes('/api/education/personnel/records/personnel-1/assignments') && method === 'GET') {
      return fulfillJson(route, {
        items: [
          {
            id: 'assignment-1',
            assignment_code: 'ASN-2026-001',
            assignment_type: 'diriginte',
            assignment_title: 'Dirigentie clasa a VII-a B',
            status: 'activ',
            assigned_on: '2025-09-01',
            ended_on: '',
            weekly_hours: 1,
          },
        ],
        total: 1,
        page: 1,
        pageSize: 10,
      });
    }

    if (/\/api\/education\/portfolios\/records\/portfolio-1\/transfer-summary(?:\?.*)?$/.test(url) && method === 'GET') {
      const lastStatus = portfolioTransferState;
      const transferStatus = lastStatus === 'trimis' ? 'sent' : lastStatus === 'receptionat' || lastStatus === 'inchis' ? 'received' : 'prepared';
      return fulfillJson(route, {
        portfolio: {
          id: 'portfolio-1',
          portfolio_code: 'PORT-2026-001',
          owner_name: 'Prof. Elena Ionescu',
          owner_role: 'Profesor limba romana',
          school_year: '2025-2026',
          transfer_status: transferStatus,
        },
        completeness: {
          total_documents: 5,
          portfolio_documents: 3,
          personnel_documents: 2,
          sensitive_documents: 1,
          total_checklist_items: 5,
          mandatory_checklist_items: 4,
          completed_checklist_items: 3,
          partial_checklist_items: 1,
          missing_checklist_items: 1,
          reviewing_checklist_items: 0,
          opis_entries: 4,
          custody_events: 2,
          review_events: 1,
          valorification_events: 3,
          ready_for_review: false,
          ready_for_transfer: false,
          blockers: ['mandatory_checklist_pending'],
        },
        transfer: {
          total_events: 1,
          prepared_events: lastStatus === 'pregatit' ? 1 : 0,
          sent_events: lastStatus === 'trimis' ? 1 : 0,
          received_events: lastStatus === 'receptionat' ? 1 : 0,
          closed_events: lastStatus === 'inchis' ? 1 : 0,
          current_direction: 'Scoala Gimnaziala Demonstrativa -> Scoala Gimnaziala Partenera',
          last_transfer: {
            id: 'transfer-1',
            transfer_code: 'TRF-2026-0001',
            transfer_type: 'detasare',
            source_institution: 'Scoala Gimnaziala Demonstrativa',
            destination_institution: 'Scoala Gimnaziala Partenera',
            status: lastStatus,
            handover_on: '2026-06-24',
            received_on: lastStatus === 'pregatit' || lastStatus === 'trimis' ? '' : '2026-06-25',
            handover_by: 'Secretariat Demo',
            received_by: lastStatus === 'pregatit' || lastStatus === 'trimis' ? '' : 'Secretariat Partener',
          },
        },
        mobility: {
          matched_cases: 1,
          active_cases: lastStatus === 'inchis' ? 0 : 1,
          transfer_cases: 0,
          detachment_cases: 1,
          restriction_cases: 0,
          current_unit_mentions: 1,
          destination_mentions: 1,
        },
        valorification: {
          total_events: portfolioValorifications.length,
          open_events: portfolioValorifications.filter((item) => ['planificat', 'in_pregatire', 'transmis', 'validat'].includes(item.status)).length,
          completed_events: portfolioValorifications.filter((item) => item.status === 'finalizat').length,
          linked_evaluations: 1,
          linked_mobility: 1,
          linked_merit: 1,
          last_event: portfolioValorifications[1],
          scopes: [
            { scope: 'evaluare_profesionala', total: 1, open: 0, completed: 1 },
            { scope: 'mobilitate', total: 1, open: 1, completed: 0 },
            { scope: 'gradatie_merit', total: 1, open: 1, completed: 0 },
          ],
        },
        readiness: {
          ready_to_request: false,
          ready_to_send: lastStatus === 'pregatit',
          ready_to_confirm: lastStatus === 'trimis',
          ready_to_close: lastStatus === 'receptionat',
          blockers: [],
        },
      });
    }

    if (/\/api\/education\/portfolios\/records\/portfolio-1\/valorifications(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        items: portfolioValorifications,
        total: portfolioValorifications.length,
        page: 1,
        pageSize: 10,
      });
    }

    if (/\/api\/education\/portfolios\/records\/portfolio-1\/valorifications\/valorification-\d+(?:\?.*)?$/.test(url) && method === 'GET') {
      const item = portfolioValorifications.find((candidate) => url.includes(candidate.id));
      return fulfillJson(route, item ?? portfolioValorifications[0]);
    }

    if (/\/api\/education\/portfolios\/records\/portfolio-1\/transfers\/transfer-1\/advance(?:\?.*)?$/.test(url) && method === 'POST') {
      const body = route.request().postDataJSON() as Record<string, unknown>;
      const action = String(body['action'] ?? '');
      const status = action === 'mark_sent' ? 'trimis' : action === 'confirm_received' ? 'receptionat' : 'inchis';
      portfolioTransferState = status as typeof portfolioTransferState;
      return fulfillJson(route, {
        id: 'transfer-1',
        portfolio_id: 'portfolio-1',
        transfer_code: 'TRF-2026-0001',
        transfer_type: 'detasare',
        source_institution: 'Scoala Gimnaziala Demonstrativa',
        destination_institution: 'Scoala Gimnaziala Partenera',
        status,
        handover_on: '2026-06-24',
        received_on: action === 'mark_sent' ? '' : '2026-06-25',
        handover_by: 'Secretariat Demo',
        received_by: action === 'mark_sent' ? '' : 'Secretariat Partener',
      });
    }

    if (/\/api\/education\/portfolios\/records\/portfolio-1\/opis\/regenerate(?:\?.*)?$/.test(url) && method === 'POST') {
      return fulfillJson(route, {
        status: 'ok',
        portfolio_id: 'portfolio-1',
        regenerated_entries: 4,
        checked_by: 'Director Demo',
      });
    }

    if (/\/api\/education\/portfolios\/records\/portfolio-1\/transfers(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        items: [
          {
            id: 'transfer-1',
            portfolio_id: 'portfolio-1',
            transfer_code: 'TRF-2026-0001',
            transfer_type: 'detasare',
            source_institution: 'Scoala Gimnaziala Demonstrativa',
            destination_institution: 'Scoala Gimnaziala Partenera',
            status: portfolioTransferState,
            handover_on: '2026-06-24',
            received_on: portfolioTransferState === 'pregatit' || portfolioTransferState === 'trimis' ? '' : '2026-06-25',
            handover_by: 'Secretariat Demo',
            received_by: portfolioTransferState === 'pregatit' || portfolioTransferState === 'trimis' ? '' : 'Secretariat Partener',
          },
        ],
        total: 1,
        page: 1,
        pageSize: 10,
      });
    }

    if (/\/api\/education\/portfolios\/records\/portfolio-1(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        id: 'portfolio-1',
        portfolio_code: 'PORT-2026-001',
        owner_name: 'Prof. Elena Ionescu',
        owner_role: 'Profesor limba romana',
        school_year: '2025-2026',
        status: portfolioSelfServiceStatus,
        section_count: 5,
        last_updated_on: '2026-06-24',
        retention_until: '2027-08-31',
        transfer_status: 'prepared',
        authenticity_declared: true,
        consent_captured: true,
        custodian: 'Secretariat',
        institution_id: 'school-1',
        notes: 'Portofoliu demonstrativ.',
      });
    }

    if (/\/api\/education\/portfolios\/records\/portfolio-1(?:\?.*)?$/.test(url) && method === 'PATCH') {
      const body = route.request().postDataJSON() as Record<string, unknown>;
      portfolioSelfServiceStatus = String(body['status'] ?? 'draft') as typeof portfolioSelfServiceStatus;
      return fulfillJson(route, {
        id: 'portfolio-1',
        portfolio_code: 'PORT-2026-001',
        owner_name: String(body['owner_name'] ?? 'Prof. Elena Ionescu'),
        owner_role: String(body['owner_role'] ?? 'Profesor limba romana'),
        school_year: String(body['school_year'] ?? '2025-2026'),
        status: portfolioSelfServiceStatus,
        section_count: Number(body['section_count'] ?? 5),
        last_updated_on: String(body['last_updated_on'] ?? '2026-06-24'),
        retention_until: String(body['retention_until'] ?? '2027-08-31'),
        transfer_status: String(body['transfer_status'] ?? 'prepared'),
        authenticity_declared: Boolean(body['authenticity_declared'] ?? true),
        consent_captured: Boolean(body['consent_captured'] ?? true),
        custodian: String(body['custodian'] ?? 'Secretariat'),
        institution_id: 'school-1',
        notes: String(body['notes'] ?? 'Portofoliu demonstrativ.'),
      });
    }

    if (url.includes('/api/education/portfolios/records') && method === 'GET' && !url.includes('/documents') && !url.includes('/checklist') && !url.includes('/opis') && !url.includes('/custody') && !url.includes('/transfers') && !url.includes('/reviews') && !url.includes('/filters') && !/\/api\/education\/portfolios\/records\/[^/?]+$/.test(url)) {
      return fulfillJson(route, {
        items: [
          {
            id: 'portfolio-1',
            portfolio_code: 'PORT-2026-001',
            owner_name: 'Prof. Elena Ionescu',
            owner_role: 'Profesor limba romana',
            school_year: '2025-2026',
            status: portfolioSelfServiceStatus,
            section_count: 5,
            transfer_status: 'prepared',
            last_updated_on: '2026-06-24',
            retention_until: '2027-08-31',
          },
        ],
        total: 1,
        page: 1,
        pageSize: url.includes('pageSize=100') ? 100 : 20,
      });
    }

    if (url.includes('/api/education/governance/bodies') && method === 'GET' && !/\/api\/education\/governance\/bodies\/[^/?]+$/.test(url)) {
      return fulfillJson(route, {
        items: [
          {
            id: '2025-2026__ca',
            school_year: '2025-2026',
            organism: 'ca',
            active_members: 5,
            voting_members: 5,
            chairperson_covered: true,
            secretary_covered: true,
            expired_mandates: 0,
            total_meetings: 2,
            scheduled_meetings: 1,
            held_meetings: 1,
            published_meetings: 0,
            latest_meeting_on: '2026-06-30',
            readiness_status: 'complet',
          },
          {
            id: '2025-2026__cp',
            school_year: '2025-2026',
            organism: 'cp',
            active_members: 0,
            voting_members: 0,
            chairperson_covered: false,
            secretary_covered: false,
            expired_mandates: 0,
            total_meetings: 0,
            scheduled_meetings: 0,
            held_meetings: 0,
            published_meetings: 0,
            latest_meeting_on: '',
            readiness_status: 'critic',
          },
        ],
        total: 2,
        page: 1,
        pageSize: url.includes('pageSize=100') ? 100 : 20,
      });
    }

    if (/\/api\/education\/committees\/records\/committee-1\/completeness-summary(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        committee: {
          id: 'committee-1',
          committee_code: 'COM-EVAL-2026-001',
          school_year: '2025-2026',
          committee_type: 'evaluare_personal_didactic',
          title: 'Comisia temporara de evaluare a personalului didactic',
          status: 'active',
          decision_reference: 'DEC-2026-CP-014',
          starts_on: '2026-06-01',
          ends_on: '2026-08-31',
          evaluation_scope: true,
        },
        membership: {
          active_members: 2,
          voting_members: 2,
          chairperson_covered: true,
          secretary_covered: true,
          member_names: ['Director Demo', 'Secretariat Demo'],
        },
        readiness: {
          ready_for_operation: true,
          blockers: [],
        },
      });
    }

    if (/\/api\/education\/committees\/records\/committee-1\/members(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        items: [
          {
            id: 'committee-member-1',
            committee_id: 'committee-1',
            full_name: 'Director Demo',
            role_name: 'Director',
            member_type: 'presedinte',
            voting_right: true,
            status: 'active',
            appointed_on: '2026-06-01',
            released_on: '',
          },
          {
            id: 'committee-member-2',
            committee_id: 'committee-1',
            full_name: 'Secretariat Demo',
            role_name: 'Secretar unitate',
            member_type: 'secretar',
            voting_right: true,
            status: 'active',
            appointed_on: '2026-06-01',
            released_on: '',
          },
        ],
        total: 2,
        page: 1,
        pageSize: 10,
      });
    }

    if (/\/api\/education\/committees\/records\/committee-1(?:\?.*)?$/.test(url) && method === 'GET') {
      return fulfillJson(route, {
        id: 'committee-1',
        committee_code: 'COM-EVAL-2026-001',
        school_year: '2025-2026',
        committee_type: 'evaluare_personal_didactic',
        title: 'Comisia temporara de evaluare a personalului didactic',
        status: 'active',
        decision_reference: 'DEC-2026-CP-014',
        starts_on: '2026-06-01',
        ends_on: '2026-08-31',
        evaluation_scope: true,
        institution_id: 'school-1',
        notes: 'Comisie activa.',
      });
    }

    if (url.includes('/api/education/governance/memberships') && method === 'GET' && !/\/api\/education\/governance\/memberships\/[^/?]+$/.test(url)) {
      return fulfillJson(route, {
        items: [
          {
            id: 'membership-1',
            school_year: '2025-2026',
            organism: 'ca',
            full_name: 'Director Demo',
            role_name: 'Presedinte',
            mandate_from: '2025-09-01',
            mandate_to: '2026-08-31',
            voting_right: true,
            status: 'activ',
          },
          {
            id: 'membership-2',
            school_year: '2025-2026',
            organism: 'ca',
            full_name: 'Secretariat Demo',
            role_name: 'Secretar',
            mandate_from: '2025-09-01',
            mandate_to: '2026-08-31',
            voting_right: true,
            status: 'activ',
          },
        ],
        total: 2,
        page: 1,
        pageSize: url.includes('pageSize=100') ? 100 : 20,
      });
    }

    if (url.includes('/api/education/committees/records') && method === 'GET' && !url.includes('/members') && !/\/api\/education\/committees\/records\/[^/?]+$/.test(url)) {
      return fulfillJson(route, {
        items: [
          {
            id: 'committee-1',
            committee_code: 'COM-EVAL-2026-001',
            school_year: '2025-2026',
            committee_type: 'evaluare_personal_didactic',
            title: 'Comisia temporara de evaluare a personalului didactic',
            status: 'active',
            decision_reference: 'DEC-2026-CP-014',
            starts_on: '2026-06-01',
            ends_on: '2026-08-31',
            evaluation_scope: true,
          },
        ],
        total: 1,
        page: 1,
        pageSize: url.includes('pageSize=100') ? 100 : 20,
      });
    }

    if (url.includes('/api/education/regulations/records') && method === 'GET' && !url.includes('/versions') && !url.includes('/workflow') && !url.includes('/filters') && !/\/api\/education\/regulations\/records\/[^/?]+$/.test(url)) {
      return fulfillJson(route, {
        items: [
          {
            id: 'regulation-1',
            regulation_code: 'REGL-2026-001',
            school_year: '2025-2026',
            regulation_type: 'roi',
            title: 'Regulament de ordine interioara',
            status: 'consultation',
            approval_status: 'consulted',
            review_due_on: '2026-09-01',
          },
        ],
        total: 1,
        page: 1,
        pageSize: url.includes('pageSize=100') ? 100 : 20,
      });
    }

    if (url.includes('/api/education/personnel/records') && method === 'GET' && !url.includes('/assignments') && !url.includes('/disciplinary-cases') && !url.includes('/file-documents') && !url.includes('/access-events') && !url.includes('/filters') && !/\/api\/education\/personnel\/records\/[^/?]+$/.test(url)) {
      return fulfillJson(route, {
        items: [
          {
            id: 'personnel-1',
            employee_code: 'PER-2026-0001',
            full_name: 'Prof. Elena Ionescu',
            role_title: 'Profesor limba romana',
            employment_type: 'titular',
            status: 'active',
            evaluation_status: 'finalized',
            has_portfolio: true,
          },
        ],
        total: 1,
        page: 1,
        pageSize: url.includes('pageSize=100') ? 100 : 20,
      });
    }

    if (url.includes('/api/education/') && method === 'GET') {
      if (url.includes('/dashboard')) {
        return fulfillJson(route, { stats: {} });
      }
      return fulfillJson(route, {
        items: [],
        total: 0,
        page: 1,
        pageSize: 20,
      });
    }

    if (url.includes('/api/education/') && ['POST', 'PATCH', 'DELETE'].includes(method)) {
      return fulfillJson(route, { status: 'ok' });
    }

    return route.continue();
  });
}

function fulfillJson(route: Route, body: unknown) {
  return route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(body),
  });
}

interface MockSessionResponse {
  userId: string;
  name: string;
  email: string;
  roles: string[];
  permissions: string[];
  modules: Array<{ code: string; active: boolean }>;
  institutionId: string;
  institutionName: string;
}

function buildSessionResponse(override?: Partial<MockSessionResponse>) {
  const userId = override?.userId ?? 'director-1';
  const name = override?.name ?? 'Director Demo';
  const email = override?.email ?? 'director@example.test';
  const roles = override?.roles ?? ['director'];

  return {
    user: {
      id: userId,
      sub: userId,
      name,
      email,
      locale: 'ro',
      roles,
    },
    institution_id: override?.institutionId ?? 'school-1',
    institution_name: override?.institutionName ?? 'Scoala Gimnaziala Demonstrativa',
    permissions:
      override?.permissions ?? [
        'registratura.read',
        'registratura.manage',
        'registratura.write',
        'workflow.read',
        'workflow.write',
        'earchiva.read',
        'earchiva.write',
        'education.read',
        'education.governance.read',
        'education.governance.manage',
        'education.managerial.read',
        'education.managerial.manage',
        'education.regulations.read',
        'education.regulations.manage',
        'education.personnel.manage',
        'education.evaluations.manage',
        'education.declarations.read',
        'education.declarations.manage',
        'education.mobility.read',
        'education.mobility.manage',
        'education.gradatii.read',
        'education.gradatii.manage',
        'education.portfolios.manage',
        'education.personnel.read',
        'education.personnel.files.read',
        'education.personnel.files.manage',
        'education.personnel.access.read',
        'education.personnel.access.manage',
        'education.evaluations.read',
        'education.portfolios.read',
      ],
    modules: override?.modules ?? [
      { code: 'education', active: true },
      { code: 'registratura', active: true },
      { code: 'workflow', active: true },
      { code: 'earchiva', active: true },
      { code: 'admin', active: true },
    ],
    authentication: ['oidc'],
    gdpr_capabilities: [],
  };
}

function defaultCockpit(): MockDirectorCockpitResponse {
  return {
    school_year: '2025-2026',
    institution_id: 'school-1',
    governance: {
      total_meetings: 12,
      scheduled_meetings: 3,
      meetings_without_minute: 2,
      meetings_without_vote: 1,
      published_resolutions: 8,
    },
    portfolios: {
      total_records: 42,
      draft_records: 8,
      review_records: 5,
      returned_records: 2,
      validated_records: 27,
      transfer_in_progress: 1,
    },
    evaluations: {
      total_records: 35,
      submitted_records: 10,
      reviewed_records: 15,
      contested_records: 2,
      approved_records: 8,
      communicated_documents: 12,
    },
    managerial: {
      total_dossiers: 9,
      draft_dossiers: 2,
      review_dossiers: 3,
      approved_dossiers: 4,
      published_documents: 11,
      workflow_open_steps: 5,
    },
    personnel: {
      total_records: 55,
      active_records: 51,
      portfolio_enabled: 40,
      evaluation_pending: 6,
      mobility_cases: 2,
    },
    compliance: {
      total_requirements: 25,
      implemented_requirements: 10,
      partial_requirements: 9,
      pending_publications: 4,
      anonymization_pending: 1,
    },
    alerts: [
      {
        id: 'evaluations-contested',
        title: 'Evaluari contestate',
        summary: 'Exista 2 evaluari aflate in etapa de contestatie.',
        status: 'contested',
        route: '/education/personnel',
        priority: 1,
      },
      {
        id: 'publications-pending',
        title: 'Publicari restante',
        summary: 'Exista 4 publicari educationale care nu sunt inca finalizate.',
        status: 'pending',
        route: '/education/compliance',
        priority: 2,
      },
    ],
    recommended_links: [
      { key: 'governance', label: 'Sedinte si hotarari', route: '/education/governance' },
      { key: 'portfolios', label: 'Portofolii profesionale', route: '/education/portfolio' },
      { key: 'personnel', label: 'Evaluari si personal', route: '/education/personnel' },
    ],
  };
}
