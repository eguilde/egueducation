import { expect, it, vi } from "vitest";
import { createSchoolClassesApi } from "./school-classes-api";
import type { ContractClient } from "../../api/client";

const result = <T,>(data: T) => Promise.resolve({ data, response: new Response() });
it("uses generated literal School operations for list and lifecycle", async () => {
  const GET=vi.fn().mockResolvedValue({data:{items:[],total:0,page:1,pageSize:20},response:new Response()}); const DELETE=vi.fn().mockImplementation(() => result({id:"class-1"}));
  const api=createSchoolClassesApi({GET,DELETE,POST:vi.fn(),PATCH:vi.fn()} as unknown as ContractClient);
  await api.classes.list({page:2,pageSize:20,sort:"class_name",direction:"asc",filters:{class_name:"IV A"}});
  expect(GET).toHaveBeenCalledWith("/api/education/classes",expect.objectContaining({params:{query:expect.objectContaining({page:2,pageSize:20,sort:"class_name","filter.class_name":"IV A"})}}));
  await api.classes.lifecycle("class-1");
  expect(DELETE).toHaveBeenCalledWith("/api/education/classes/{classID}",{params:{path:{classID:"class-1"}}});
  await api.assignmentOptions({kind:"teachers",q:"ionescu",page:1,pageSize:25});
  expect(GET).toHaveBeenCalledWith("/api/education/classes/assignment-options",{params:{query:{kind:"teachers",q:"ionescu",page:1,pageSize:25}}});
});

it("serializes every non-class header filter and contracted sort through its generated operation", async () => {
  const GET=vi.fn().mockResolvedValue({data:{items:[],total:0,page:1,pageSize:20},response:new Response()});
  const api=createSchoolClassesApi({GET,DELETE:vi.fn(),POST:vi.fn(),PATCH:vi.fn()} as unknown as ContractClient);
  await api.students.list({page:1,pageSize:20,sort:"birth_date",direction:"desc",filters:{birth_date:"2018-01-01"}});
  await api.enrolments.list({page:1,pageSize:20,sort:"enrolled_until",direction:"asc",filters:{enrolled_until:"2026-06-01"}});
  await api.homerooms.list({page:1,pageSize:20,sort:"class_name",direction:"asc",filters:{class_name:"IV A",assigned_until:"2026-06-01"}});
  expect(GET).toHaveBeenCalledWith("/api/education/students",expect.objectContaining({params:{query:expect.objectContaining({sort:"birth_date","filter.birth_date":"2018-01-01"})}}));
  expect(GET).toHaveBeenCalledWith("/api/education/class-enrolments",expect.objectContaining({params:{query:expect.objectContaining({sort:"enrolled_until","filter.enrolled_until":"2026-06-01"})}}));
  expect(GET).toHaveBeenCalledWith("/api/education/homeroom-assignments",expect.objectContaining({params:{query:expect.objectContaining({sort:"class_name","filter.class_name":"IV A","filter.assigned_until":"2026-06-01"})}}));
});
