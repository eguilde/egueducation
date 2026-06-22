import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';

import {
  CancelRegistraturaDocumentRequest,
  BatchCreateRegistraturaDocumentRequest,
  CreateRegistraturaRegistryRequest,
  CreateRegistraturaDocumentAttachmentRequest,
  CreateRegistraturaDocumentRequest,
  CreateRegistraturaDocumentVersionRequest,
  ExportRegistraturaDocumentsRequest,
  PagedResponse,
  RegistraturaDocumentAttachment,
  RegistraturaDocument,
  RegistraturaDocumentFilters,
  RegistraturaDocumentVersion,
  RegistraturaRegistry,
  UpdateRegistraturaDocumentRequest,
  UpdateRegistraturaRegistryRequest,
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

  updateDocument(documentId: string, payload: UpdateRegistraturaDocumentRequest) {
    return this.http.patch<RegistraturaDocument>(`/api/registratura/documents/${documentId}`, payload);
  }

  cancelDocument(documentId: string, payload: CancelRegistraturaDocumentRequest) {
    return this.http.post<RegistraturaDocument>(`/api/registratura/documents/${documentId}/cancel`, payload);
  }

  batchCreateDocuments(payload: BatchCreateRegistraturaDocumentRequest) {
    return this.http.post<RegistraturaDocument[]>('/api/registratura/documents/batch', payload);
  }

  exportDocuments(payload: ExportRegistraturaDocumentsRequest) {
    return this.http.post('/api/registratura/documents/export-pdf', payload, { responseType: 'blob' as const });
  }

  registries() {
    return this.http.get<RegistraturaRegistry[]>('/api/registratura/registre');
  }

  registry(registryId: number) {
    return this.http.get<RegistraturaRegistry>(`/api/registratura/registre/${registryId}`);
  }

  defaultRegistry() {
    return this.http.get<RegistraturaRegistry>('/api/registratura/registre/default');
  }

  createRegistry(payload: CreateRegistraturaRegistryRequest) {
    return this.http.post<RegistraturaRegistry>('/api/registratura/registre', payload);
  }

  updateRegistry(registryId: number, payload: UpdateRegistraturaRegistryRequest) {
    return this.http.patch<RegistraturaRegistry>(`/api/registratura/registre/${registryId}`, payload);
  }

  deleteRegistry(registryId: number) {
    return this.http.delete<void>(`/api/registratura/registre/${registryId}`);
  }

  setDefaultRegistry(registryId: number) {
    return this.http.patch<RegistraturaRegistry>(`/api/registratura/registre/${registryId}/set-default`, {});
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
