# go-worker-event
go-worker-event


# table

    CREATE TABLE order_clearance_reconciliacion (
        id 						BIGSERIAL		NOT NULL,
        fk_clearance_id			BIGINT			NULL,
        fk_order_id				BIGINT			NULL,
        transaction_id			VARCHAR(100) 	NULL,
        clearance_status 		VARCHAR(100) 	NULL,
        clearance_currency 		VARCHAR(100) 	NULL,
        clearance_amount 		DECIMAL(10,2) 	NOT null DEFAULT 0,
        order_status 			VARCHAR(100) 	NULL,
        order_currency 			VARCHAR(100) 	NULL,
        order_amount 			DECIMAL(10,2) 	NOT null DEFAULT 0,
        reconciliacion_type 	VARCHAR(100) 	NULL,
        reconciliacion_status 	VARCHAR(100) 	NULL,
        reconciliacion_currency VARCHAR(100) 	NULL,
        reconciliacion_amount 	DECIMAL(10,2) 	NOT null DEFAULT 0,
        created_at 				timestamptz 	NOT NULL,
        updated_at 				timestamptz  	NULL,
        CONSTRAINT order_clearance_reconciliacion_pkey PRIMARY KEY (id)
    );
