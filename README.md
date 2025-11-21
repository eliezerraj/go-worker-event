# go-worker-event
go-worker-event


# table

    CREATE TABLE order_clearance_reconciliacion (
        id 						BIGSERIAL		NOT NULL,
        fk_clearance_id			BIGINT		NULL,
        fk_order_id				BIGINT		NULL,
        transaction_id			VARCHAR(100) 	NULL,
        clearance_type 			VARCHAR(100) 	NOT NULL,
        clearance_status 		VARCHAR(100) 	NOT NULL,
        clearance_currency 		VARCHAR(100) 	NOT NULL,
        clearance_amount 		DECIMAL(10,2) 	NOT null DEFAULT 0,
        order_currency 			VARCHAR(100) 	NOT NULL,
        order_amount 			DECIMAL(10,2) 	NOT null DEFAULT 0,
        reconciliacion_currency VARCHAR(100) 	NOT NULL,
        reconciliacion_amount 	DECIMAL(10,2) 	NOT null DEFAULT 0,
        reconciliacion_status 	VARCHAR(100) 	NOT NULL,
        created_at 				timestamptz 	NOT NULL,
        updated_at 				timestamptz  	NULL,
        CONSTRAINT order_clearance_reconciliacion_pkey PRIMARY KEY (id)
    );

