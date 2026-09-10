import type { ContractClient } from "../../api/client";
import type { components } from "../../api/generated";
import type { SchoolClassesApi, SchoolPage, SchoolQuery } from "./SchoolClassesWorkspace";

const requireData = async <T,>(result: Promise<{ data?: T; error?: unknown }>): Promise<T> => {
  const resolved = await result;
  if (resolved.error || !resolved.data) throw new Error("school_classes_contract_request_failed");
  return resolved.data;
};
const classSorts = ["class_code", "class_name", "school_year", "grade_level", "active"] as const;
const studentSorts = ["student_code", "first_name", "last_name", "status", "birth_date"] as const;
const enrolmentSorts = ["student_name", "class_name", "enrolled_from", "enrolled_until", "status"] as const;
const homeroomSorts = ["teacher_name", "class_name", "assigned_from", "assigned_until"] as const;
const query = <TSort extends string>(value: SchoolQuery, allowedSorts: readonly TSort[]) => {
  if (value.sort && !allowedSorts.includes(value.sort as TSort)) throw new Error("school_classes_sort_not_in_contract");
  return { page:value.page, pageSize:value.pageSize, sort:value.sort as TSort | undefined, direction:value.direction, ...Object.fromEntries(Object.entries(value.filters).filter(([, entry]) => entry.trim()).map(([key, entry]) => [`filter.${key}`, entry])) };
};
const page = <T,>(value: { items: T[]; total: number; page: number; pageSize: number }): SchoolPage<T> => value;

/** Only generated literal OpenAPI operations are used below; there is no path/fetch fallback. */
export function createSchoolClassesApi(client: ContractClient): SchoolClassesApi {
  return {
    classes: {
      list: async input => page(await requireData(client.GET("/api/education/classes", { params: { query: query(input, classSorts) } })) as components["schemas"]["EducationPageOfSchoolClass"]),
      detail: id => requireData(client.GET("/api/education/classes/{classID}", { params: { path: { classID:id } } })),
      create: input => requireData(client.POST("/api/education/classes", { body: input as components["schemas"]["CreateSchoolClassRequest"] })),
      update: (id,input) => requireData(client.PATCH("/api/education/classes/{classID}", { params: { path:{classID:id} }, body:input as components["schemas"]["CreateSchoolClassRequest"] })),
      lifecycle: id => requireData(client.DELETE("/api/education/classes/{classID}", { params: { path:{classID:id} } })),
    },
    students: {
      list: async input => page(await requireData(client.GET("/api/education/students", { params: { query: query(input, studentSorts) } })) as components["schemas"]["EducationPageOfSchoolStudent"]),
      detail: id => requireData(client.GET("/api/education/students/{studentID}", { params: { path:{studentID:id} } })),
      create: input => requireData(client.POST("/api/education/students", { body:input as components["schemas"]["CreateSchoolStudentRequest"] })),
      update: (id,input) => requireData(client.PATCH("/api/education/students/{studentID}", { params: { path:{studentID:id} }, body:input as components["schemas"]["CreateSchoolStudentRequest"] })),
      lifecycle: id => requireData(client.DELETE("/api/education/students/{studentID}", { params: { path:{studentID:id} } })),
    },
    enrolments: {
      list: async input => page(await requireData(client.GET("/api/education/class-enrolments", { params: { query: query(input, enrolmentSorts) } })) as components["schemas"]["EducationPageOfSchoolEnrolment"]),
      detail: id => requireData(client.GET("/api/education/class-enrolments/{enrolmentID}", { params: { path:{enrolmentID:id} } })),
      create: input => requireData(client.POST("/api/education/class-enrolments", { body:input as components["schemas"]["CreateSchoolEnrolmentRequest"] })),
      update: (id,input) => requireData(client.PATCH("/api/education/class-enrolments/{enrolmentID}", { params: { path:{enrolmentID:id} }, body:input as components["schemas"]["CreateSchoolEnrolmentRequest"] })),
      lifecycle: id => requireData(client.DELETE("/api/education/class-enrolments/{enrolmentID}", { params: { path:{enrolmentID:id} } })),
    },
    homerooms: {
      list: async input => page(await requireData(client.GET("/api/education/homeroom-assignments", { params: { query: query(input, homeroomSorts) } })) as components["schemas"]["EducationPageOfSchoolHomeroomAssignment"]),
      detail: id => requireData(client.GET("/api/education/homeroom-assignments/{assignmentID}", { params: { path:{assignmentID:id} } })),
      create: input => requireData(client.POST("/api/education/homeroom-assignments", { body:input as components["schemas"]["CreateSchoolHomeroomAssignmentRequest"] })),
      update: (id,input) => requireData(client.PATCH("/api/education/homeroom-assignments/{assignmentID}", { params: { path:{assignmentID:id} }, body:input as components["schemas"]["CreateSchoolHomeroomAssignmentRequest"] })),
      lifecycle: id => requireData(client.DELETE("/api/education/homeroom-assignments/{assignmentID}", { params: { path:{assignmentID:id} } })),
    },
    assignmentOptions: async input => page(await requireData(client.GET("/api/education/classes/assignment-options", { params: { query: { kind: input.kind, q: input.q, page: input.page, pageSize: input.pageSize } } })) as components["schemas"]["EducationPageOfSchoolAssignmentOption"]),
  } as SchoolClassesApi;
}
