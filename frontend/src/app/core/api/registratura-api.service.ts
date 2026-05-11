import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import {
  CreateRegistraturaDocumentAttachmentRequest,
  CreateRegistraturaDocumentRequest,
  CreateRegistraturaDocumentVersionRequest,
  PagedResponse,
  RegistraturaDocumentAttachment,
  RegistraturaDocument,
  RegistraturaDocumentFilters,
  RegistraturaDocumentVersion,
  TableQuery,
} from './api.types';

@Injectable({ providedIn: 'root' })
export class RegistraturaApiService {
  private readonly http = inject(HttpClient);

  documents(query: TableQuery) {
    let params = new HttpParams()
      .set('page', String(query.page))
      .set('pageSize', String(query.pageSize));

    if (query.sort) {
      params = params.set('sort', query.sort);
    }
    if (query.direction) {
      params = params.set('direction', query.direction);
    }
    for (const [key, value] of Object.entries(query.filters ?? {})) {
      if (value) {
        params = params.set(`filter.${key}`, value);
      }
    }

    return this.http.get<PagedResponse<RegistraturaDocument>>('/api/registratura/documents', { params });
  }

  documentFilters() {
    return this.http.get<RegistraturaDocumentFilters>('/api/registratura/documents/filters');
  }

  createDocument(payload: CreateRegistraturaDocumentRequest) {
    return this.http.post<RegistraturaDocument>('/api/registratura/documents', payload);
  }

  document(documentId: string) {
    return this.http.get<RegistraturaDocument>(`/api/registratura/documents/${documentId}`);
  }

  documentVersions(documentId: string) {
    return this.http.get<RegistraturaDocumentVersion[]>(`/api/registratura/documents/${documentId}/versions`);
  }

  createDocumentVersion(documentId: string, payload: CreateRegistraturaDocumentVersionRequest) {
    return this.http.post<RegistraturaDocumentVersion>(`/api/registratura/documents/${documentId}/versions`, payload);
  }

  documentAttachments(documentId: string) {
    return this.http.get<RegistraturaDocumentAttachment[]>(`/api/registratura/documents/${documentId}/attachments`);
  }

  createDocumentAttachment(documentId: string, payload: CreateRegistraturaDocumentAttachmentRequest) {
    return this.http.post<RegistraturaDocumentAttachment>(`/api/registratura/documents/${documentId}/attachments`, payload);
  }
}
