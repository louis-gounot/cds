//! This is a SDK of CDS to let you use the CDS API in Rust
#[macro_use]
extern crate serde_derive;
#[macro_use]
extern crate failure;
extern crate jsonwebtoken as jwt;

mod auth;
mod client;
mod error;
pub mod models;
pub mod service;

pub use client::*;
pub use error::*;
