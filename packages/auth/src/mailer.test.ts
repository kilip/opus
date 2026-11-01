import { describe, expect, it, mock } from "bun:test";
import type { SendMailOptions, Transporter } from "nodemailer";
import { createMailer, sendOTPEmail } from "./mailer";

describe("mailer", () => {
  it("falls back to console log when SMTP is not configured", async () => {
    let logged = "";
    const originalLog = console.log;
    console.log = (msg: string) => {
      logged = msg;
    };

    try {
      await sendOTPEmail({
        email: "test@example.com",
        otp: "123456",
        type: "sign-in",
      });
      expect(logged).toContain("123456");
      expect(logged).toContain("test@example.com");
    } finally {
      console.log = originalLog;
    }
  });

  it("sends email via nodemailer transporter when configured", async () => {
    let sentMailArgs: SendMailOptions | null = null;
    const fakeTransporter = {
      sendMail: mock(async (options: SendMailOptions) => {
        sentMailArgs = options;
        return { messageId: "mock-123" };
      }),
    } as unknown as Transporter;

    const mailer = createMailer({
      from: "Opus <noreply@opus.local>",
      transporter: fakeTransporter,
    });

    await mailer.sendOTPEmail({
      email: "user@example.com",
      otp: "987654",
      type: "sign-in",
    });

    expect(fakeTransporter.sendMail).toHaveBeenCalledTimes(1);
    const sent = sentMailArgs as SendMailOptions | null;
    expect(sent?.to).toBe("user@example.com");
    expect(sent?.subject).toContain("987654");
    expect(sent?.text).toContain("987654");
  });
});
