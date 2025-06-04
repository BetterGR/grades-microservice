package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	gpb "github.com/BetterGR/grades-microservice/protos"
	ms "github.com/TekClinic/MicroService-Lib"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"k8s.io/klog/v2"
)

const (
	connectionProtocol = "tcp"
	logLevelDebug      = 5
)

// DBInterface defines the interface for database operations.
type DBInterface interface {
	AddGrade(ctx context.Context, grade *gpb.SingleGrade) (*Grade, error)
	GetCourseGrades(ctx context.Context, courseID, semester string) ([]*Grade, error)
	GetStudentCourseGrades(ctx context.Context, courseID, semester, studentID string) ([]*Grade, error)
	UpdateGrade(ctx context.Context, grade *gpb.SingleGrade) (*Grade, error)
	RemoveGrade(ctx context.Context, gradeID string) error
	GetStudentSemesterGrades(ctx context.Context, studentID, semester string) ([]*Grade, error)
}

// GradesServer is the server struct still needs to implement the GradesServiceServer interface.
type GradesServer struct {
	// throws unimplemented error.
	gpb.UnimplementedGradesServiceServer
	ms.BaseServiceServer
	db     DBInterface
	Claims ms.Claims
}

// VerifyToken returns the injected Claims instead of the default.
func (s *GradesServer) VerifyToken(ctx context.Context, token string) error {
	if s.Claims != nil {
		return nil
	}

	// Default behavior.
	if _, err := s.BaseServiceServer.VerifyToken(ctx, token); err != nil {
		return fmt.Errorf("failed to verify token: %w", err)
	}

	return nil
}

func initGradesMicroserviceServer() (*GradesServer, error) {
	base, err := ms.CreateBaseServiceServer()
	if err != nil {
		return nil, fmt.Errorf("failed to create base service: %w", err)
	}

	database, err := InitializeDatabase()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return &GradesServer{
		BaseServiceServer:                base,
		UnimplementedGradesServiceServer: gpb.UnimplementedGradesServiceServer{},
		db:                               database,
	}, nil
}

// GetCourseGrades returns all students grades for a specific course for a specific semester.
func (s *GradesServer) GetCourseGrades(ctx context.Context,
	req *gpb.GetCourseGradesRequest,
) (*gpb.GetCourseGradesResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received request for course grades", "course_id", req.GetCourseID(),
		"semester", req.GetSemester())

	// get course grades.
	grades, err := s.db.GetCourseGrades(ctx, req.GetCourseID(), req.GetSemester())
	if err != nil {
		return nil, fmt.Errorf("failed to get course grades: %w", err)
	}

	return &gpb.GetCourseGradesResponse{
		Grades: s.createGradesResponse(grades),
	}, nil
}

// GetStudentCourseGrades returns all grades for a specific student in a specific course for a specific semester.
func (s *GradesServer) GetStudentCourseGrades(ctx context.Context,
	req *gpb.GetStudentCourseGradesRequest,
) (*gpb.GetStudentCourseGradesResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received request for student course grades", "course_id", req.GetCourseID(),
		"semester", req.GetSemester(), "student_id", req.GetStudentID())

	// get student course grades.
	grades, err := s.db.GetStudentCourseGrades(ctx, req.GetCourseID(), req.GetSemester(), req.GetStudentID())
	if err != nil {
		return nil, fmt.Errorf("failed to get student course grades: %w", err)
	}

	return &gpb.GetStudentCourseGradesResponse{
		Grades: s.createGradesResponse(grades),
	}, nil
}

// AddSingleGrade adds a single grade for a specific student in a specific course for a specific semester.
func (s *GradesServer) AddSingleGrade(ctx context.Context,
	req *gpb.AddSingleGradeRequest,
) (*gpb.AddSingleGradeResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received request for add single grade", "course_id", req.GetGrade().GetCourseID(),
		"semester", req.GetGrade().GetSemester(), "student_id", req.GetGrade().GetStudentID())

	// add grade.
	if _, err := s.db.AddGrade(ctx, req.GetGrade()); err != nil {
		return nil, fmt.Errorf("failed to add single grade: %w", err)
	}

	return &gpb.AddSingleGradeResponse{Grade: req.GetGrade()}, nil
}

// UpdateSingleGrade updates a single grade for a specific student in a specific course for a specific semester.
func (s *GradesServer) UpdateSingleGrade(ctx context.Context,
	req *gpb.UpdateSingleGradeRequest,
) (*gpb.UpdateSingleGradeResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received request for update single grade", "course_id", req.GetGrade().GetCourseID(),
		"semester", req.GetGrade().GetSemester(), "student_id", req.GetGrade().GetStudentID())

	// update grade.
	updatedGrade, err := s.db.UpdateGrade(ctx, req.GetGrade())
	if err != nil {
		return nil, fmt.Errorf("failed to update single grade: %w", err)
	}

	// updated grade.
	grade := &gpb.SingleGrade{
		GradeID:    updatedGrade.GradeID,
		StudentID:  updatedGrade.StudentID,
		CourseID:   updatedGrade.CourseID,
		Semester:   updatedGrade.Semester,
		GradeType:  updatedGrade.GradeType,
		ItemID:     updatedGrade.ItemID,
		GradeValue: updatedGrade.GradeValue,
		GradedBy:   updatedGrade.GradedBy,
		Comments:   updatedGrade.Comments,
	}

	return &gpb.UpdateSingleGradeResponse{Grade: grade}, nil
}

// RemoveSingleGrade removes a single grade for a specific student in a specific course for a specific semester.
func (s *GradesServer) RemoveSingleGrade(ctx context.Context,
	req *gpb.RemoveSingleGradeRequest,
) (*gpb.RemoveSingleGradeResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received request to remove a single grade", "grade_id", req.GetGradeID())

	if err := s.db.RemoveGrade(ctx, req.GetGradeID()); err != nil {
		return nil, fmt.Errorf("failed to remove single grade: %w", err)
	}

	return &gpb.RemoveSingleGradeResponse{}, nil
}

// GetStudentSemesterGrades returns all grades for a specific student for a specific semester.
func (s *GradesServer) GetStudentSemesterGrades(ctx context.Context,
	req *gpb.GetStudentSemesterGradesRequest,
) (*gpb.GetStudentSemesterGradesResponse, error) {
	if err := s.VerifyToken(ctx, req.GetToken()); err != nil {
		return nil, fmt.Errorf("authentication failed: %w",
			status.Error(codes.Unauthenticated, err.Error()))
	}

	logger := klog.FromContext(ctx)
	logger.V(logLevelDebug).Info("Received request for student semester grades",
		"semester", req.GetSemester(), "student_id", req.GetStudentID())

	// get student semester grades.
	grades, err := s.db.GetStudentSemesterGrades(ctx, req.GetStudentID(), req.GetSemester())
	if err != nil {
		return nil, fmt.Errorf("failed to get student semester grades: %w", err)
	}

	return &gpb.GetStudentSemesterGradesResponse{
		Grades: s.createGradesResponse(grades),
	}, nil
}

func (s *GradesServer) createGradesResponse(grades []*Grade) []*gpb.SingleGrade {
	gradesResponse := make([]*gpb.SingleGrade, 0, len(grades))
	for _, grade := range grades {
		gradesResponse = append(gradesResponse, &gpb.SingleGrade{
			GradeID:    grade.GradeID,
			StudentID:  grade.StudentID,
			CourseID:   grade.CourseID,
			Semester:   grade.Semester,
			GradeType:  grade.GradeType,
			ItemID:     grade.ItemID,
			GradeValue: grade.GradeValue,
			GradedBy:   grade.GradedBy,
			Comments:   grade.Comments,
		})
	}

	return gradesResponse
}

// main server function.
func main() {
	// init klog
	klog.InitFlags(nil)
	flag.Parse()

	if err := godotenv.Load(); err != nil {
    klog.Warning("Warning: No .env file loaded, proceeding with environment variables only")
}

	// Initialize the server.
	server, err := initGradesMicroserviceServer()
	if err != nil {
		klog.Fatalf("Failed to initialize server: %v", err)
	}

	// create a listener.
	address := "localhost:" + os.Getenv("GRPC_PORT")

	lis, err := net.Listen(connectionProtocol, address)
	if err != nil {
		klog.Error("Failed to listen", "error", err)
	}

	// create a grpc server.
	grpcServer := grpc.NewServer()
	gpb.RegisterGradesServiceServer(grpcServer, server)
	klog.V(logLevelDebug).Info("Grades server is running on port " + os.Getenv("GRPC_PORT"))
	// serve the grpc server.
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
