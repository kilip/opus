import nodemailer, { type Transporter } from "nodemailer";
import { authEnv } from "./env";

export interface SendOTPEmailParams {
  email: string;
  otp: string;
  type: string;
}

export interface MailerOptions {
  from?: string;
  transporter?: Transporter;
}

export function createMailer(options?: MailerOptions) {
  const from = options?.from ?? authEnv.EMAIL_FROM;

  let transporter: Transporter | null = options?.transporter ?? null;
  if (!transporter && authEnv.SMTP_HOST) {
    transporter = nodemailer.createTransport({
      host: authEnv.SMTP_HOST,
      port: authEnv.SMTP_PORT,
      secure: authEnv.SMTP_SECURE,
      auth:
        authEnv.SMTP_USER && authEnv.SMTP_PASSWORD
          ? {
              user: authEnv.SMTP_USER,
              pass: authEnv.SMTP_PASSWORD,
            }
          : undefined,
    });
  }

  async function sendOTPEmail({ email, otp, type }: SendOTPEmailParams): Promise<void> {
    if (!transporter) {
      console.log(`[AUTH OTP] ${type} code for ${email}: ${otp}`);
      return;
    }

    const subject = `Your verification code: ${otp}`;
    const text = `Your ${type} one-time password (OTP) is: ${otp}\n\nThis code will expire shortly.`;
    const html = `
      <div style="font-family: sans-serif; max-width: 480px; margin: 0 auto; padding: 24px;">
        <h2 style="margin-bottom: 16px;">Verification Code</h2>
        <p style="color: #444; font-size: 16px;">Use the code below to complete your ${type}:</p>
        <div style="background-color: #f4f4f5; padding: 16px; border-radius: 8px; text-align: center; margin: 24px 0;">
          <span style="font-size: 32px; font-weight: bold; letter-spacing: 4px; font-family: monospace;">${otp}</span>
        </div>
        <p style="color: #666; font-size: 14px;">If you didn't request this code, you can safely ignore this email.</p>
      </div>
    `;

    await transporter.sendMail({
      from,
      to: email,
      subject,
      text,
      html,
    });
  }

  return {
    sendOTPEmail,
    get transporter() {
      return transporter;
    },
  };
}

export const defaultMailer = createMailer();
export const sendOTPEmail = defaultMailer.sendOTPEmail;
