import { describe, it, expect } from "vitest";
import { getOption, hasOption } from "@bufbuild/protobuf";
import { file_user } from "../gen/quadsmith/user_pb";
import { file_auth } from "../gen/quadsmith/auth_pb";
import { rules } from "../gen/quadsmith/_validate_pb";

describe("Protobuf Regex Validation Build-Time Check (TypeScript)", () => {
  it("verifies that all Protobuf validation regex patterns compile successfully in JavaScript RegExp", () => {
    let verifiedCount = 0;

    for (const file of [file_user, file_auth]) {
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
