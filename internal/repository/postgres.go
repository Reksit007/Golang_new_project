package repository

import (
	"context"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/artem/project/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, req domain.CreateSubscriptionRequest) (domain.Subscription, error) {
	startMonth, err := domain.ParseMonth(req.StartDate)
	if err != nil {
		return domain.Subscription{}, err
	}

	var endMonth *time.Time
	if req.EndDate != "" {
		parsedEndMonth, err := domain.ParseMonth(req.EndDate)
		if err != nil {
			return domain.Subscription{}, err
		}
		endMonth = &parsedEndMonth
	}

	query := `
		INSERT INTO subscriptions (service_name, price, user_id, start_month, end_month)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, service_name, price, user_id, start_month, end_month, created_at, updated_at
	`

	return scanSubscription(r.db.QueryRow(ctx, query, req.ServiceName, req.Price, req.UserID, startMonth, endMonth))
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM subscriptions WHERE id = $1`, id)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *Repository) List(ctx context.Context, filter domain.ListSubscriptionsFilter) ([]domain.Subscription, error) {
	builder := sq.
		Select("id", "service_name", "price", "user_id", "start_month", "end_month", "created_at", "updated_at").
		From("subscriptions").
		PlaceholderFormat(sq.Dollar).
		OrderBy("created_at DESC")

	if filter.UserID != "" {
		builder = builder.Where(sq.Eq{"user_id": filter.UserID})
	}

	if filter.ServiceName != "" {
		builder = builder.Where("service_name ILIKE ?", "%"+filter.ServiceName+"%")
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.Subscription, 0)
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, sub)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *Repository) Get(ctx context.Context, id string) (domain.Subscription, error) {
	query := `
		SELECT id, service_name, price, user_id, start_month, end_month, created_at, updated_at
		FROM subscriptions
		WHERE id = $1
	`

	return scanSubscription(r.db.QueryRow(ctx, query, id))
}

func (r *Repository) Update(ctx context.Context, id string, req domain.UpdateSubscriptionRequest) (domain.Subscription, error) {
	current, err := r.Get(ctx, id)
	if err != nil {
		return domain.Subscription{}, err
	}

	if req.ServiceName != nil {
		current.ServiceName = *req.ServiceName
	}
	if req.Price != nil {
		current.Price = *req.Price
	}
	if req.UserID != nil {
		current.UserID = *req.UserID
	}
	if req.StartDate != nil {
		current.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		current.EndDate = *req.EndDate
	}

	startMonth, err := domain.ParseMonth(current.StartDate)
	if err != nil {
		return domain.Subscription{}, err
	}

	var endMonth *time.Time
	if current.EndDate != "" {
		parsedEndMonth, err := domain.ParseMonth(current.EndDate)
		if err != nil {
			return domain.Subscription{}, err
		}
		endMonth = &parsedEndMonth
	}

	query := `
		UPDATE subscriptions
		SET service_name = $2,
		    price = $3,
		    user_id = $4,
		    start_month = $5,
		    end_month = $6,
		    updated_at = now()
		WHERE id = $1
		RETURNING id, service_name, price, user_id, start_month, end_month, created_at, updated_at
	`

	return scanSubscription(r.db.QueryRow(
		ctx,
		query,
		id,
		current.ServiceName,
		current.Price,
		current.UserID,
		startMonth,
		endMonth,
	))
}

func (r *Repository) Summary(ctx context.Context, filter domain.SummaryFilter) (int, error) {
	builder := sq.
		Select().
		Column(`
			COALESCE(SUM(
				price * (
					(EXTRACT(YEAR FROM AGE(LEAST(COALESCE(end_month, ?), ?), GREATEST(start_month, ?)))::int * 12)
					+ EXTRACT(MONTH FROM AGE(LEAST(COALESCE(end_month, ?), ?), GREATEST(start_month, ?)))::int
					+ 1
				)
			), 0)::int
		`, filter.PeriodEnd, filter.PeriodEnd, filter.PeriodStart, filter.PeriodEnd, filter.PeriodEnd, filter.PeriodStart).
		From("subscriptions").
		PlaceholderFormat(sq.Dollar).
		Where("start_month <= ?", filter.PeriodEnd).
		Where("COALESCE(end_month, ?) >= ?", filter.PeriodEnd, filter.PeriodStart)

	if filter.UserID != "" {
		builder = builder.Where(sq.Eq{"user_id": filter.UserID})
	}

	if filter.ServiceName != "" {
		builder = builder.Where("service_name ILIKE ?", "%"+filter.ServiceName+"%")
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return 0, err
	}

	var total int
	if err := r.db.QueryRow(ctx, query, args...).Scan(&total); err != nil {
		return 0, err
	}

	return total, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSubscription(row scanner) (domain.Subscription, error) {
	var sub domain.Subscription
	var start time.Time
	var end *time.Time

	err := row.Scan(
		&sub.ID,
		&sub.ServiceName,
		&sub.Price,
		&sub.UserID,
		&start,
		&end,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	)
	if err != nil {
		return domain.Subscription{}, err
	}

	sub.StartDate = start.Format(domain.MonthLayout)
	if end != nil {
		sub.EndDate = end.Format(domain.MonthLayout)
	}

	return sub, nil
}
