// frontend/types/auth.js
// Type definitions based on backend SafeUser and inputs

/**
 * @typedef {Object} User
 * @property {string} id
 * @property {string} email
 * @property {string} first_name
 * @property {string} last_name
 * @property {string|Date} date_of_birth
 * @property {string} [avatar_path]
 * @property {string} [nickname]
 * @property {string} [about_me]
 * @property {string} profile_visibility
 * @property {string|Date} created_at
 * @property {string|Date} updated_at
 */

/**
 * @typedef {Object} LoginInput
 * @property {string} email
 * @property {string} password
 */

/**
 * @typedef {Object} RegisterInput
 * @property {string} email
 * @property {string} password
 * @property {string} first_name
 * @property {string} last_name
 * @property {string} date_of_birth  // YYYY-MM-DD format
 * @property {string} [avatar_path]
 * @property {string} [nickname]
 * @property {string} [about_me]
 */

