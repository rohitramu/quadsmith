// @vitest-environment node
import { describe, it, expect } from "vitest";
import { getOption, hasOption } from "@bufbuild/protobuf";
import { file_quadsmith_api_user } from "@quadsmith/shared/gen/quadsmith/api/user_pb";
import { file_quadsmith_api_auth } from "@quadsmith/shared/gen/quadsmith/api/auth_pb";
import { rules } from "@quadsmith/shared/gen/quadsmith/validate/validate_pb";

describe("Protobuf Regex Validation Build-Time Check (TypeScript)", () => {
  it("verifies that all Protobuf validation regex patterns compile successfully in JavaScript RegExp", () => {
    let verifiedCount = 0;

    for (const file of [file_quadsmith_api_user, file_quadsmith_api_auth]) {
      for (const msg of file.messages) {
        for (const field of msg.fields) {
          if (hasOption(field, rules)) {
            const fieldRules = getOption(field, rules);
            if (fieldRules.string?.pattern) {
              const pattern = fieldRules.string.pattern;
              const fieldPath = `${msg.name}.${field.name}`;

              // Build-time check: Pattern must compile under JS RegExp without throwing SyntaxError
              expect(
                () => new RegExp(pattern),
                `Field ${fieldPath} regex pattern "${pattern}" must compile in JavaScript RegExp`,
              ).not.toThrow();

              const compiled = new RegExp(pattern);
              expect(compiled.test("fpv_pilot")).toBe(true);
              verifiedCount++;
            }
          }
        }
      }
    }

    expect(verifiedCount).toBeGreaterThanOrEqual(2);
  });
});
