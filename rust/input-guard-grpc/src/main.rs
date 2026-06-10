use std::net::SocketAddr;
use std::time::{Instant, SystemTime, UNIX_EPOCH};

use input_guard::validate_input;
use tonic::{transport::Server, Request, Response, Status};
use tonic_health::server::health_reporter;
use tracing::{info, warn};
use tracing_subscriber::EnvFilter;

pub mod validator {
    pub mod v1 {
        tonic::include_proto!("validator.v1");
    }
}

use validator::v1::validator_server::{Validator, ValidatorServer};
use validator::v1::{
    ExecutionMetadata, HealthRequest, HealthResponse, RuleResult, ValidateInputRequest,
    ValidateInputResponse,
};

#[derive(Debug)]
struct ValidatorService {
    start_time: SystemTime,
}

impl ValidatorService {
    fn new() -> Self {
        Self {
            start_time: SystemTime::now(),
        }
    }

    fn build_response(&self, req: ValidateInputRequest) -> ValidateInputResponse {
        let started = Instant::now();
        let result = validate_input(&req.input);
        let elapsed_us = started.elapsed().as_micros() as i64;
        let valid = result.is_ok();
        let reason = result.err().unwrap_or_default();

        let rule_result = RuleResult {
            rule_id: "INPUT-GUARD-CLI-COMPAT".to_string(),
            rule_name: "Compatibilidad con reglas locales".to_string(),
            passed: valid,
            severity: if valid { "INFO" } else { "CRITICAL" }.to_string(),
            message: reason.clone(),
            evaluation_time_us: elapsed_us,
        };

        ValidateInputResponse {
            valid,
            reason,
            rule_results: vec![rule_result],
            metadata: Some(ExecutionMetadata {
                total_time_us: elapsed_us,
                rules_evaluated: 1,
                validator_version: env!("CARGO_PKG_VERSION").to_string(),
                trace_id: req.correlation_id,
            }),
        }
    }
}

#[tonic::async_trait]
impl Validator for ValidatorService {
    async fn validate_input(
        &self,
        request: Request<ValidateInputRequest>,
    ) -> Result<Response<ValidateInputResponse>, Status> {
        let req = request.into_inner();
        let input_len = req.input.len();
        let correlation_id = req.correlation_id.clone();
        let response = self.build_response(req);

        if response.valid {
            info!(correlation_id, input_len, "input válido");
        } else {
            warn!(correlation_id, input_len, reason = %response.reason, "input rechazado");
        }

        Ok(Response::new(response))
    }

    async fn health(
        &self,
        _request: Request<HealthRequest>,
    ) -> Result<Response<HealthResponse>, Status> {
        let uptime_seconds = self.start_time.elapsed().unwrap_or_default().as_secs() as i64;

        Ok(Response::new(HealthResponse {
            status: validator::v1::health_response::Status::Healthy as i32,
            version: env!("CARGO_PKG_VERSION").to_string(),
            uptime_seconds,
        }))
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    tracing_subscriber::fmt()
        .with_env_filter(EnvFilter::from_default_env())
        .json()
        .init();

    let addr = std::env::var("INPUT_GUARD_GRPC_ADDR")
        .or_else(|_| std::env::var("INPUT_GUARD_ADDR"))
        .unwrap_or_else(|_| "0.0.0.0:50051".to_string());
    let socket_addr: SocketAddr = addr.parse()?;

    let (mut reporter, health_service) = health_reporter();
    let service = ValidatorService::new();
    reporter
        .set_serving::<ValidatorServer<ValidatorService>>()
        .await;

    let started_at = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default()
        .as_secs();
    info!(addr, started_at, "iniciando input-guard gRPC");

    Server::builder()
        .add_service(health_service)
        .add_service(ValidatorServer::new(service))
        .serve(socket_addr)
        .await?;

    Ok(())
}
